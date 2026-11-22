// SolarisPKN LiveTools - MultiChat native OBS source (Windows)
// Zero embedded browsers. One hidden OBS GDI+ text renderer per visible MultiChat source.
// Network access: GET 127.0.0.1:<port>/obs/messages?limit=N only.
#include <obs-module.h>
#include <obs-frontend-api.h>
#include <windows.h>
#include <winhttp.h>

#include <algorithm>
#include <atomic>
#include <condition_variable>
#include <deque>
#include <chrono>
#include <cctype>
#include <cstdlib>
#include <cstdint>
#include <mutex>
#include <string>
#include <thread>
#include <utility>

OBS_DECLARE_MODULE()

namespace {


// OBS frontend event bridge: no stream keys, session cookies or RTMP URLs ever sent.
struct OBSEventBridge {
    std::mutex mutex;
    std::condition_variable cv;
    std::deque<std::pair<std::string,std::string>> queued;
    std::thread worker;
    bool running = true;
    bool streaming = false;
    std::string service = "unknown";
};
OBSEventBridge eventBridge;

std::string obsServiceCategory()
{
    // Only the public OBS service label is inspected. Do not read "key" / "server".
    obs_service_t *service = obs_frontend_get_streaming_service();
    if (!service) return "unknown";
    obs_data_t *settings = obs_service_get_settings(service);
    std::string label;
    if (settings) {
        label = obs_data_get_string(settings, "service");
        obs_data_release(settings);
    }
    if (label.empty()) label = obs_service_get_type(service);
    if (label.size() > 96) label.resize(96);
    std::transform(label.begin(), label.end(), label.begin(),
                   [](unsigned char c) { return static_cast<char>(std::tolower(c)); });
    for (const char *key : {"youtube", "twitch", "facebook", "kick", "tiktok"})
        if (label.find(key) != std::string::npos) return key;
    return "unknown";
}

void sendOBSEvent(const std::string &kind, const std::string &service)
{
    wchar_t token[256] = {};
    const DWORD chars = GetEnvironmentVariableW(L"MULTICHAT_OBS_EVENT_TOKEN", token, 256);
    if (chars < 32 || chars >= 256) return;
    const std::wstring secret(token, chars);
    if (secret.find_first_of(L"\r\n") != std::wstring::npos) return;

    unsigned port = 8766;
    wchar_t portText[12] = {};
    DWORD portLength = GetEnvironmentVariableW(L"MULTICHAT_PORT", portText, 12);
    if (portLength > 0 && portLength < 12) {
        wchar_t *end = nullptr;
        const unsigned long value = wcstoul(portText, &end, 10);
        if (end && *end == 0 && value >= 1024 && value <= 65535)
            port = static_cast<unsigned>(value);
    }
    const std::string body = "{\"event\":\"" + kind + "\",\"service\":\"" + service + "\"}";
    const std::wstring headers =
        L"Authorization: Bearer " + secret +
        L"\r\nX-SolarisPKN-Action: obs\r\nContent-Type: application/json\r\n";
    HINTERNET session = WinHttpOpen(L"SolarisPKN-OBS-EventBridge/0.1",
                                    WINHTTP_ACCESS_TYPE_NO_PROXY, nullptr, nullptr, 0);
    if (!session) return;
    WinHttpSetTimeouts(session, 800, 800, 900, 1000);
    HINTERNET connection = WinHttpConnect(session, L"127.0.0.1",
                                          static_cast<INTERNET_PORT>(port), 0);
    if (connection) {
        HINTERNET req = WinHttpOpenRequest(connection, L"POST",
                                           L"/api/obs/event", nullptr,
                                           WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, 0);
        if (req) {
            WinHttpSendRequest(req, headers.c_str(), DWORD(-1),
                               (LPVOID)body.data(), DWORD(body.size()),
                               DWORD(body.size()), 0);
            WinHttpReceiveResponse(req, nullptr);
            WinHttpCloseHandle(req);
        }
        WinHttpCloseHandle(connection);
    }
    WinHttpCloseHandle(session);
}

void eventWorker()
{
    for (;;) {
        std::pair<std::string,std::string> item;
        {
            std::unique_lock<std::mutex> lk(eventBridge.mutex);
            eventBridge.cv.wait_for(lk, std::chrono::seconds(15), []{
                return !eventBridge.running || !eventBridge.queued.empty();
            });
            if (!eventBridge.running) break;
            if (!eventBridge.queued.empty()) {
                item = std::move(eventBridge.queued.front());
                eventBridge.queued.pop_front();
            } else if (eventBridge.streaming) {
                // Heartbeat: catches the case where OBS started before Go MultiChat.
                item = {"started", eventBridge.service};
            } else {
                continue;
            }
        }
        sendOBSEvent(item.first,item.second);
    }
}

void onOBSStreamEvent(enum obs_frontend_event event, void *)
{
    if (event != OBS_FRONTEND_EVENT_STREAMING_STARTED &&
        event != OBS_FRONTEND_EVENT_STREAMING_STOPPED) return;
    const std::string state = event == OBS_FRONTEND_EVENT_STREAMING_STARTED
                                  ? "started" : "stopped";
    const std::string service = obsServiceCategory();
    {
        std::lock_guard<std::mutex> lk(eventBridge.mutex);
        // Do not queue unbounded events if an external process is unavailable.
        if (eventBridge.queued.size() >= 8) eventBridge.queued.pop_front();
        eventBridge.streaming = state == "started";
        eventBridge.service = service;
        eventBridge.queued.emplace_back(state,service);
    }
    eventBridge.cv.notify_one();
}

constexpr uint32_t kDefaultWidth = 740;
constexpr uint32_t kDefaultHeight = 460;
constexpr size_t kMaximumResponse = 128 * 1024;

struct ChatSource {
    obs_source_t *text = nullptr;
    std::atomic<bool> running{true};
    std::atomic<unsigned> port{8765};
    std::atomic<int> lines{8};
    std::atomic<int> pollMs{900};
    std::thread worker;
    std::mutex pendingMutex;
    std::string pending;
    bool changed = false;
    uint32_t width = kDefaultWidth;
    uint32_t height = kDefaultHeight;
    int fontSize = 29;
    std::string lastRendered;
};

std::string clean(std::string input, size_t maxBytes)
{
    for (auto &ch : input) {
        unsigned char c = static_cast<unsigned char>(ch);
        if (c < 32 || c == 127) ch = ' ';
    }
    if (input.size() <= maxBytes) return input;
    size_t cut = maxBytes;
    // Never cut through a multi-byte UTF-8 code point.
    while (cut > 0 && cut < input.size() &&
           (static_cast<unsigned char>(input[cut]) & 0xC0) == 0x80) {
        --cut;
    }
    input.resize(cut);
    input += "...";
    return input;
}

std::string fetch(unsigned port, int limit)
{
    // WinHTTP direct loopback request: no proxy, no remote URLs, no credentials.
    HINTERNET session = WinHttpOpen(L"SolarisPKN-MultiChat-OBS/0.1",
                                    WINHTTP_ACCESS_TYPE_NO_PROXY, nullptr, nullptr, 0);
    if (!session) return {};
    WinHttpSetTimeouts(session, 1000, 1000, 1200, 1200);
    HINTERNET connection = WinHttpConnect(session, L"127.0.0.1",
                                           static_cast<INTERNET_PORT>(port), 0);
    std::string result;
    if (connection) {
        std::wstring path = L"/obs/messages?limit=" + std::to_wstring(limit);
        HINTERNET request = WinHttpOpenRequest(connection, L"GET", path.c_str(),
                                               nullptr, WINHTTP_NO_REFERER,
                                               WINHTTP_DEFAULT_ACCEPT_TYPES, 0);
        if (request) {
            // Separate read-only OBS token; never stored in scene settings or logged.
            wchar_t secret[256] = {};
            const DWORD size = GetEnvironmentVariableW(L"MULTICHAT_OBS_TOKEN", secret, 256);
            std::wstring authHeader;
            if (size >= 32 && size < 256) {
                const std::wstring token(secret, size);
                if (token.find_first_of(L"\r\n") == std::wstring::npos)
                    authHeader = L"Authorization: Bearer " + token + L"\r\n";
            }
            if (WinHttpSendRequest(request,
                                   authHeader.empty() ? WINHTTP_NO_ADDITIONAL_HEADERS : authHeader.c_str(),
                                   authHeader.empty() ? 0 : static_cast<DWORD>(-1),
                                   WINHTTP_NO_REQUEST_DATA, 0, 0, 0) &&
                WinHttpReceiveResponse(request, nullptr)) {
                DWORD status = 0;
                DWORD length = sizeof(status);
                if (WinHttpQueryHeaders(request, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
                                        WINHTTP_HEADER_NAME_BY_INDEX, &status, &length,
                                        WINHTTP_NO_HEADER_INDEX) && status == 200) {
                    char buffer[4096];
                    DWORD n = 0;
                    while (WinHttpReadData(request, buffer, sizeof(buffer), &n) && n != 0) {
                        if (result.size() + n > kMaximumResponse) {
                            result.clear();
                            break;
                        }
                        result.append(buffer, n);
                    }
                }
            }
            WinHttpCloseHandle(request);
        }
        WinHttpCloseHandle(connection);
    }
    WinHttpCloseHandle(session);
    return result;
}

std::string renderMessages(const std::string &response, int limit)
{
    if (response.empty() || response.front() != '[') return {};
    // OBS's JSON parser expects a JSON object. Wrap the bounded array.
    obs_data_t *root = obs_data_create_from_json(("{\"items\":" + response + "}").c_str());
    if (!root) return {};
    obs_data_array_t *items = obs_data_get_array(root, "items");
    std::string output;
    if (items) {
        const size_t count = obs_data_array_count(items);
        const size_t start = count > static_cast<size_t>(limit) ? count - limit : 0;
        for (size_t i = start; i < count; ++i) {
            obs_data_t *item = obs_data_array_item(items, i);
            if (!item) continue;
            std::string platform = clean(obs_data_get_string(item, "platform"), 24);
            std::string user = clean(obs_data_get_string(item, "user"), 48);
            std::string text = clean(obs_data_get_string(item, "text"), 190);
            obs_data_release(item);
            if (text.empty()) continue;
            // One text texture for the complete chat, not one renderer per message.
            output += "[" + platform + "] " + user + ": " + text + "\n";
        }
        obs_data_array_release(items);
    }
    obs_data_release(root);
    return output;
}

void pollingLoop(ChatSource *source)
{
    while (source->running.load()) {
        const int lines = source->lines.load();
        const std::string response = fetch(source->port.load(), lines);
        if (!response.empty()) {
            const std::string next = renderMessages(response, lines);
            std::lock_guard<std::mutex> lock(source->pendingMutex);
            if (source->pending != next) {
                source->pending = next;
                source->changed = true;
            }
        }
        // Interruptible sleep, so deleting a source does not stall OBS shutdown.
        const int ms = source->pollMs.load();
        for (int elapsed = 0; elapsed < ms && source->running.load(); elapsed += 50) {
            std::this_thread::sleep_for(std::chrono::milliseconds(50));
        }
    }
}

void updateTextSource(ChatSource *source, const std::string &value)
{
    if (!source->text) return;
    obs_data_t *settings = obs_source_get_settings(source->text);
    if (!settings) return;

    obs_data_set_string(settings, "text", value.c_str());
    obs_data_set_bool(settings, "read_from_file", false);
    obs_data_set_bool(settings, "extents", true);
    obs_data_set_bool(settings, "extents_wrap", true);
    obs_data_set_int(settings, "extents_cx", source->width);
    obs_data_set_int(settings, "extents_cy", source->height);
    obs_data_set_int(settings, "bk_opacity", 0);
    obs_data_set_int(settings, "opacity", 100);
    obs_data_set_int(settings, "color", 0xFFFFFF);
    obs_data_set_bool(settings, "outline", true);
    obs_data_set_int(settings, "outline_size", 2);
    obs_data_set_int(settings, "outline_color", 0x050505);
    obs_data_t *font = obs_data_create();
    obs_data_set_string(font, "face", "Segoe UI");
    obs_data_set_int(font, "size", source->fontSize);
    obs_data_set_string(font, "style", "Regular");
    obs_data_set_obj(settings, "font", font);
    obs_data_release(font);

    obs_source_update(source->text, settings);
    obs_data_release(settings);
}

const char *getName(void *)
{
    return "SolarisPKN MultiChat (Native - No Browser)";
}

void getDefaults(obs_data_t *settings)
{
    obs_data_set_default_int(settings, "port", 8765);
    obs_data_set_default_int(settings, "lines", 8);
    obs_data_set_default_int(settings, "poll_ms", 900);
    obs_data_set_default_int(settings, "width", kDefaultWidth);
    obs_data_set_default_int(settings, "height", kDefaultHeight);
    obs_data_set_default_int(settings, "font_size", 29);
}

obs_properties_t *getProperties(void *)
{
    auto *properties = obs_properties_create();
    obs_properties_add_int(properties, "port", "MultiChat local port", 1024, 65535, 1);
    obs_properties_add_int(properties, "lines", "Visible messages", 1, 16, 1);
    obs_properties_add_int(properties, "poll_ms", "Refresh interval (ms)", 300, 5000, 100);
    obs_properties_add_int(properties, "width", "Overlay width", 320, 1920, 10);
    obs_properties_add_int(properties, "height", "Overlay height", 150, 1080, 10);
    obs_properties_add_int(properties, "font_size", "Font size", 14, 64, 1);
    return properties;
}

void update(void *data, obs_data_t *settings)
{
    auto *source = static_cast<ChatSource *>(data);
    source->port.store(static_cast<unsigned>(std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "port")), 1024, 65535)));
    source->lines.store(std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "lines")), 1, 16));
    source->pollMs.store(std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "poll_ms")), 300, 5000));
    source->width = static_cast<uint32_t>(std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "width")), 320, 1920));
    source->height = static_cast<uint32_t>(std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "height")), 150, 1080));
    source->fontSize = std::clamp<int>(
        static_cast<int>(obs_data_get_int(settings, "font_size")), 14, 64);
    // Changes applied in OBS video_tick, avoiding calls from the network worker.
}

void *create(obs_data_t *settings, obs_source_t *)
{
    auto *source = new ChatSource();
    source->text = obs_source_create_private("text_gdiplus", nullptr, nullptr);
    if (!source->text) {
        blog(LOG_ERROR, "[MultiChat] OBS Text (GDI+) source unavailable.");
        delete source;
        return nullptr;
    }
    update(source, settings);
    updateTextSource(source, "MultiChat: esperando mensajes...");
    source->worker = std::thread(pollingLoop, source);
    return source;
}

void destroy(void *data)
{
    auto *source = static_cast<ChatSource *>(data);
    if (!source) return;
    source->running.store(false);
    if (source->worker.joinable()) source->worker.join();
    if (source->text) obs_source_release(source->text);
    delete source;
}

void videoTick(void *data, float)
{
    auto *source = static_cast<ChatSource *>(data);
    std::string text;
    bool changed = false;
    {
        std::lock_guard<std::mutex> lock(source->pendingMutex);
        if (source->changed) {
            text = source->pending;
            source->changed = false;
            changed = true;
        }
    }
    if (changed && text != source->lastRendered) {
        source->lastRendered = text;
        updateTextSource(source, text);
    }
}

void videoRender(void *data, gs_effect_t *)
{
    auto *source = static_cast<ChatSource *>(data);
    if (source->text) obs_source_video_render(source->text);
}

uint32_t getWidth(void *data)
{
    return static_cast<ChatSource *>(data)->width;
}

uint32_t getHeight(void *data)
{
    return static_cast<ChatSource *>(data)->height;
}

} // namespace

extern "C" bool obs_module_load(void)
{
    obs_source_info info = {};
    info.id = "solarispkn_multichat_native";
    info.type = OBS_SOURCE_TYPE_INPUT;
    info.output_flags = OBS_SOURCE_VIDEO | OBS_SOURCE_CUSTOM_DRAW;
    info.get_name = getName;
    info.create = create;
    info.destroy = destroy;
    info.update = update;
    info.get_defaults = getDefaults;
    info.get_properties = getProperties;
    info.video_tick = videoTick;
    info.video_render = videoRender;
    info.get_width = getWidth;
    info.get_height = getHeight;
    obs_register_source(&info);
    eventBridge.worker = std::thread(eventWorker);
    obs_frontend_add_event_callback(onOBSStreamEvent, nullptr);
    // OBS may already be streaming when the module is (re)loaded.
    if (obs_frontend_streaming_active())
        onOBSStreamEvent(OBS_FRONTEND_EVENT_STREAMING_STARTED, nullptr);
    blog(LOG_INFO, "[MultiChat] Native source + OBS streaming event bridge registered; no browser used.");
    return true;
}

extern "C" void obs_module_unload(void)
{
    obs_frontend_remove_event_callback(onOBSStreamEvent, nullptr);
    {
        std::lock_guard<std::mutex> lock(eventBridge.mutex);
        eventBridge.running = false;
        eventBridge.queued.clear();
    }
    eventBridge.cv.notify_one();
    if (eventBridge.worker.joinable()) eventBridge.worker.join();
}
