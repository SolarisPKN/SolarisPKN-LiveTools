# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Weekly opt-in staging of official GitHub releases. Never installs while running."""
from __future__ import annotations

import hashlib
import json
import os
import re
import threading
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlparse
from urllib.request import HTTPRedirectHandler, Request, build_opener, urlopen

from i18n import get_language, t
from version import APP_VERSION, DEFAULT_RELEASE_REPOSITORY, MODULE_PATH

REPO = DEFAULT_RELEASE_REPOSITORY
API = f"https://api.github.com/repos/{REPO}/releases?per_page=100"
PERIOD = timedelta(days=7)
RETRY = timedelta(hours=24)
MAX_BYTES = 150 * 1024 * 1024
TAG = re.compile(r"^(?:solarispkn-)?anticensor-v(\d+)\.(\d+)\.(\d+)$", re.I)
SEMVER = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)$", re.I)
DIGEST = re.compile(r"^sha256:([0-9a-f]{64})$", re.I)
TRUSTED_HOSTS = frozenset(("github.com", "release-assets.githubusercontent.com",
                           "objects.githubusercontent.com"))


def app_folder() -> Path:
    return Path(os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")) / "SolarisPKN" / "AntiCensor"


def state_path() -> Path:
    return app_folder() / "updates.json"


def pending_path() -> Path:
    return app_folder() / "pending_update.json"


def _load(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
        return data if isinstance(data, dict) else {}
    except (OSError, ValueError, TypeError):
        return {}


def _save(path: Path, data: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temp = path.with_name(path.name + ".tmp")
    try:
        temp.write_text(json.dumps(data, ensure_ascii=False, indent=2), encoding="utf-8")
        os.replace(temp, path)
    finally:
        temp.unlink(missing_ok=True)


def load_state() -> dict:
    return _load(state_path())


def save_state(data: dict) -> None:
    # Discard legacy configurable repo. The canonical repo is never editable.
    data.pop("repo", None)
    _save(state_path(), data)


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def version_triplet(value: str) -> tuple[int, int, int] | None:
    m = SEMVER.fullmatch(str(value).strip())
    return tuple(int(v) for v in m.groups()) if m else None


def module_release_version(value: str) -> tuple[int, int, int] | None:
    m = TAG.fullmatch(str(value).strip())
    return tuple(int(v) for v in m.groups()) if m else None


def release_asset_name(version: tuple[int, int, int]) -> str:
    return "SolarisPKN-AntiCensor-Setup-v" + ".".join(map(str, version)) + ".exe"


def auto_enabled() -> bool:
    return load_state().get("auto_updates") is True


def pending_update() -> dict:
    return _load(pending_path())


def set_auto_enabled(enabled: bool) -> None:
    data = load_state()
    data["auto_updates"] = bool(enabled)
    save_state(data)
    if not enabled:
        pending = pending_update()
        if pending.get("source") == "automatic":
            pending_path().unlink(missing_ok=True)
            name = str(pending.get("file", ""))
            # Delete only our staged EXE inside the user's private updates dir.
            if Path(name).parent == app_folder() / "Updates" and Path(name).name.startswith("SolarisPKN-AntiCensor-Setup-v"):
                Path(name).unlink(missing_ok=True)


def due() -> bool:
    state = load_state()
    try:
        last = datetime.fromisoformat(state.get("checked_at", ""))
        if last.tzinfo is None:
            return True
    except (ValueError, TypeError):
        return True
    return now_utc() >= last + (PERIOD if state.get("check_success") else RETRY)


@dataclass(frozen=True)
class UpdateResult:
    status: str
    message: str
    release_url: str | None = None
    tag: str | None = None
    version: tuple[int, int, int] | None = None
    asset_url: str | None = None
    sha256: str | None = None
    size: int = 0


class TrustedRedirects(HTTPRedirectHandler):
    def redirect_request(self, request, fp, code, msg, headers, url):
        parsed = urlparse(url)
        if parsed.scheme != "https" or parsed.hostname not in TRUSTED_HOSTS:
            raise ValueError("Untrusted release redirect")
        return super().redirect_request(request, fp, code, msg, headers, url)


class UpdateChecker:
    def __init__(self):
        self.lock = threading.Lock()
        self.download_lock = threading.Lock()

    def repository(self):
        return REPO

    def due(self):
        return due()

    def check(self, force: bool = False) -> UpdateResult:
        language = get_language()
        def msg(key, **kwargs):
            return t(key, language, **kwargs)
        if not self.lock.acquire(blocking=False):
            return UpdateResult("busy", msg("update_busy"))
        try:
            if not force and (not auto_enabled() or not due()):
                return UpdateResult("not_due", msg("update_not_due"))
            success = False
            try:
                req = Request(API, headers={"User-Agent": "SolarisPKN-AntiCensor/" + APP_VERSION,
                               "Accept": "application/vnd.github+json"})
                with urlopen(req, timeout=12) as response:
                    if response.status != 200:
                        raise OSError(f"GitHub HTTP {response.status}")
                    releases = json.loads(response.read(1_000_000).decode("utf-8"))
                if not isinstance(releases, list):
                    raise ValueError("Expected GitHub Releases list")
                matches = []
                for item in releases:
                    if not isinstance(item, dict) or item.get("draft") or item.get("prerelease"):
                        continue
                    tag = str(item.get("tag_name", ""))
                    version = module_release_version(tag)
                    link = f"https://github.com/{REPO}/releases/tag/{tag}"
                    if not version or item.get("html_url") != link:
                        continue
                    expected = release_asset_name(version)
                    assets = item.get("assets", [])
                    for asset in assets if isinstance(assets, list) else ():
                        if not isinstance(asset, dict) or asset.get("name") != expected:
                            continue
                        digest = DIGEST.fullmatch(str(asset.get("digest", "")))
                        href = f"https://github.com/{REPO}/releases/download/{tag}/{expected}"
                        size = int(asset.get("size", 0))
                        if digest and asset.get("browser_download_url") == href and 1_000_000 <= size <= MAX_BYTES:
                            matches.append(UpdateResult("available", msg("update_available", tag=tag,
                                version=APP_VERSION), link, tag, version, href, digest.group(1).lower(), size))
                success = True
                current = version_triplet(APP_VERSION)
                if current is None:
                    return UpdateResult("invalid_version", msg("update_bad_local_version"))
                if not matches:
                    return UpdateResult("missing", msg("update_missing"))
                best = max(matches, key=lambda x: x.version)
                if best.version > current:
                    return best
                return UpdateResult("current", msg("update_current", version=APP_VERSION))
            except HTTPError as ex:
                if ex.code == 404:
                    success = True
                    return UpdateResult("missing", msg("update_missing"))
                return UpdateResult("offline", msg("update_offline", error=f"HTTP {ex.code}"))
            except (OSError, ValueError, URLError, TypeError) as ex:
                return UpdateResult("offline", msg("update_offline", error=ex))
            finally:
                state = load_state()
                state.update(checked_at=now_utc().isoformat(), check_success=success)
                save_state(state)
        finally:
            self.lock.release()

    def stage(self, release: UpdateResult, automatic: bool) -> UpdateResult:
        lang = get_language()
        if automatic and not auto_enabled():
            return UpdateResult("disabled", t("auto_off", lang))
        if not all((release.asset_url, release.sha256, release.version, release.size)):
            return UpdateResult("no_asset", t("update_no_package", lang))
        if not self.download_lock.acquire(blocking=False):
            return UpdateResult("busy", t("update_busy", lang))
        folder = app_folder() / "Updates"
        file = folder / release_asset_name(release.version)
        tmp = file.with_suffix(".exe.download")
        try:
            folder.mkdir(parents=True, exist_ok=True)
            digest = hashlib.sha256()
            total = 0
            opener = build_opener(TrustedRedirects())
            with opener.open(Request(release.asset_url, headers={
                    "User-Agent": "SolarisPKN-AntiCensor/" + APP_VERSION
                }), timeout=25) as response, tmp.open("wb") as output:
                final = urlparse(response.geturl())
                if final.scheme != "https" or final.hostname not in TRUSTED_HOSTS:
                    raise ValueError("Untrusted GitHub download")
                while True:
                    block = response.read(262144)
                    if not block:
                        break
                    total += len(block)
                    if total > MAX_BYTES:
                        raise ValueError("Release exceeds maximum size")
                    output.write(block)
                    digest.update(block)
            if total != release.size or digest.hexdigest() != release.sha256:
                raise ValueError("Release SHA256 or size mismatch")
            if automatic and not auto_enabled():
                return UpdateResult("disabled", t("auto_off", lang))
            os.replace(tmp, file)
            _save(pending_path(), {
                "version": ".".join(map(str, release.version)), "tag": release.tag,
                "file": str(file), "sha256": release.sha256, "bytes": total,
                "source": "automatic" if automatic else "manual",
            })
            return UpdateResult("ready", t("update_staged", lang), tag=release.tag,
                                version=release.version)
        except (OSError, URLError, ValueError, TypeError) as ex:
            return UpdateResult("download_error", t("update_download_error", lang, error=ex))
        finally:
            tmp.unlink(missing_ok=True)
            self.download_lock.release()
