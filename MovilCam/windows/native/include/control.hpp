#pragma once
// Camera demand control: broker connects to Android only while at least one
// virtual-camera application is actively using its DirectShow capture pin.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <atomic>
#include <memory>
#include <thread>
namespace movilcam {
inline constexpr wchar_t control_pipe[] = LR"(\\.\pipe\SolarisPKN_MovilCam_Control_v1)";
class ControlServer {
    struct State {
        std::atomic<int> clients{0};
        std::atomic<bool> running{true};
    };
    std::shared_ptr<State> state_ = std::make_shared<State>();
    std::thread acceptor_;
public:
    ControlServer() = default;
    ControlServer(const ControlServer&) = delete;
    ControlServer& operator=(const ControlServer&) = delete;
    ~ControlServer(){stop();}
    bool start();
    void stop();
    [[nodiscard]] int clients() const noexcept {return state_->clients.load();}
};
}
