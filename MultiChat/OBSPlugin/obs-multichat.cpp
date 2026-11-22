// SolarisPKN LiveTools - MultiChat native OBS source (Windows)
// Zero embedded browsers. One hidden OBS GDI+ text renderer per visible MultiChat source.
// Network access: GET 127.0.0.1:<port>/obs/messages?limit=N only.
#include <obs-module.h>
#include <winhttp.h>

#include <algorithm>
#include <atomic>
#include <chrono>
#include <cstdint>
#include <mutex>
#include <string>
#include <thread>
#include <utility>

OBS_DECLARE_MODULE()

namespace {

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
            if (WinHttpSendRequest(request, WINHTTP_NO_ADDITIONAL_HEADERS, 0,
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
    blog(LOG_INFO, "[MultiChat] Native source registered; no CEF/Browser Source used.");
    return true;
}
