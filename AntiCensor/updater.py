# -*- coding: utf-8 -*-
# SPDX-License-Identifier: GPL-3.0-only
"""Read-only weekly GitHub Releases check for AntiCensor module in LiveTools.

Release tags must be scoped to this module: anticensor-vMAJOR.MINOR.PATCH.
Generic LiveTools releases are ignored, so another component never triggers an
AntiCensor update. Only official GitHub links are shown; no EXE is executed.
"""
from __future__ import annotations
import json
import os
import re
import threading
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
from i18n import get_language, t
from version import APP_VERSION, DEFAULT_RELEASE_REPOSITORY

PERIOD = timedelta(days=7)
RETRY = timedelta(hours=24)
API_ROOT = "https://api.github.com/repos"
USER_AGENT = "SolarisPKN-AntiCensor-Updater/" + APP_VERSION
_REPO = re.compile(r"^[A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9_.-]{1,100}$")
_VERSION = re.compile(r"^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$", flags=re.I)
_RELEASE_TAG = re.compile(r"^anticensor-v(\d+)\.(\d+)\.(\d+)$", flags=re.I)


def state_path() -> str:
    base = os.environ.get("LOCALAPPDATA") or os.path.expanduser("~")
    return os.path.join(base, "SolarisPKN", "AntiCensor", "updates.json")


def validate_repo(repo: str) -> str:
    repo = repo.strip()
    if not _REPO.fullmatch(repo) or ".." in repo or repo.endswith("."):
        raise ValueError("Invalid GitHub repository: use owner/name.")
    return repo


def version_triplet(value: str) -> tuple[int, int, int] | None:
    m = _VERSION.fullmatch(str(value).strip())
    return tuple(int(x) for x in m.groups()) if m else None


def module_release_version(tag: str) -> tuple[int, int, int] | None:
    match = _RELEASE_TAG.fullmatch(str(tag).strip())
    return tuple(int(x) for x in match.groups()) if match else None


def now_utc() -> datetime:
    return datetime.now(timezone.utc)


def load_state() -> dict:
    try:
        with open(state_path(), encoding="utf-8") as f:
            obj = json.load(f)
        return obj if isinstance(obj, dict) else {}
    except (OSError, ValueError, TypeError):
        return {}


def save_state(data: dict) -> None:
    path = state_path()
    os.makedirs(os.path.dirname(path), exist_ok=True)
    tmp = path + ".tmp"
    try:
        with open(tmp, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
        os.replace(tmp, path)
    finally:
        if os.path.exists(tmp):
            os.remove(tmp)


@dataclass(frozen=True)
class UpdateResult:
    status: str
    message: str
    release_url: str | None = None
    tag: str | None = None


class UpdateChecker:
    def __init__(self):
        self.lock = threading.Lock()

    def repository(self) -> str:
        d = load_state()
        return validate_repo(str(d.get("repo", DEFAULT_RELEASE_REPOSITORY)))

    def set_repository(self, name: str) -> None:
        name = validate_repo(name)
        with self.lock:
            st = load_state()
            st["repo"] = name
            st.pop("checked_at", None)
            st.pop("check_success", None)
            save_state(st)

    def due(self) -> bool:
        state = load_state()
        try:
            last = datetime.fromisoformat(state.get("checked_at", ""))
            if last.tzinfo is None:
                return True
        except (ValueError, TypeError):
            return True
        cadence = PERIOD if state.get("check_success") else RETRY
        return now_utc() >= last + cadence

    def check(self, force: bool = False) -> UpdateResult:
        language = get_language()
        def text(key, **kwargs):
            return t(key, language, **kwargs)
        if not self.lock.acquire(blocking=False):
            return UpdateResult("busy", text("update_busy"))
        try:
            if not force and not self.due():
                return UpdateResult("not_due", text("update_not_due"))
            repo = self.repository()
            url = f"{API_ROOT}/{repo}/releases?per_page=100"
            req = Request(url, headers={
                "User-Agent": USER_AGENT,
                "Accept": "application/vnd.github+json",
                "X-GitHub-Api-Version": "2022-11-28",
            })
            good = False
            try:
                with urlopen(req, timeout=9) as resp:
                    if resp.status != 200:
                        raise OSError("GitHub HTTP " + str(resp.status))
                    payload = resp.read(400_000)
                releases = json.loads(payload.decode("utf-8"))
                if not isinstance(releases, list):
                    raise ValueError("GitHub releases API returned non-list JSON")
                candidates = []
                for item in releases:
                    if not isinstance(item, dict) or item.get("draft") or item.get("prerelease"):
                        continue
                    tag = str(item.get("tag_name", ""))
                    version = module_release_version(tag)
                    link = str(item.get("html_url", ""))
                    if version is None:
                        continue  # Ignore other modules' releases
                    if link != f"https://github.com/{repo}/releases/tag/{tag}":
                        continue
                    candidates.append((version, tag, link))
                good = True
                if not candidates:
                    return UpdateResult("missing", text("update_missing"))
                version, tag, link = max(candidates, key=lambda item: item[0])
                current = version_triplet(APP_VERSION)
                if current is None:
                    return UpdateResult("invalid_version", text("update_bad_local_version"))
                if version > current:
                    return UpdateResult("available",
                                        text("update_available", tag=tag, version=APP_VERSION),
                                        release_url=link, tag=tag)
                return UpdateResult("current", text("update_current", version=APP_VERSION))
            except HTTPError as e:
                if e.code == 404:
                    good = True
                    return UpdateResult("missing", text("update_missing"))
                return UpdateResult("offline", text("update_offline", error=f"HTTP {e.code}"))
            except (URLError, TimeoutError, OSError, ValueError, TypeError) as e:
                return UpdateResult("offline", text("update_offline", error=e))
            finally:
                state = load_state()
                state["checked_at"] = now_utc().isoformat()
                state["check_success"] = good
                state["repo"] = repo
                save_state(state)
        finally:
            self.lock.release()
