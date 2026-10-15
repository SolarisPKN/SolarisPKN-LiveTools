#pragma once
// Interprocess triple-buffer. All data is *NV12*, no RGB conversion in bridge.
// Producer publishes whole frames with slot seqlocks; readers never block capture.
// Consumers must copy a slot and verify the sequence did not change.
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>
#include <cstdint>
#include <cstddef>
#include <memory>
#include <string>

namespace movilcam {
inline constexpr DWORD ring_magic = 0x4d43414d; // "MCAM"
inline constexpr DWORD ring_version = 2;
inline constexpr int ring_slots = 3;
inline constexpr wchar_t ring_name[] = L"Local\\SolarisPKN_MovilCam_NV12_v2";

struct alignas(64) SlotHeader {
    volatile LONG64 sequence = 0; // odd means writing; even means stable
    LONGLONG timestamp100ns = 0;
    DWORD bytes = 0;
    DWORD reserved = 0;
    std::uint8_t padding[40]{};
};
static_assert(sizeof(SlotHeader)==64);
struct alignas(64) RingHeader {
    DWORD magic = ring_magic;
    DWORD version = ring_version;
    DWORD width = 0;
    DWORD height = 0;
    DWORD pitch = 0;
    DWORD frameBytes = 0;
    DWORD slotStride = 0;
    DWORD fps = 60; // Native DirectShow capture rate; ring protocol v2
    volatile LONG64 published = 0; // increasing publish counter
    volatile LONG64 totalFrames = 0;
    volatile LONG64 lastQpc = 0;
    volatile LONG64 dropped = 0;
};
static_assert(sizeof(RingHeader)%64 == 0);

class FrameRing {
    HANDLE mapping_ = nullptr;
    std::uint8_t* base_ = nullptr;
    RingHeader* header_ = nullptr;
    DWORD stride_ = 0;
    std::uint64_t next_ = 0;
public:
    FrameRing() = default;
    FrameRing(const FrameRing&) = delete;
    FrameRing& operator=(const FrameRing&) = delete;
    ~FrameRing() { close(); }
    bool create(DWORD width, DWORD height, DWORD fps);
    void close() noexcept;
    bool publish(const std::uint8_t* nv12, std::size_t size, LONGLONG pts100ns) noexcept;
    [[nodiscard]] const RingHeader* header() const noexcept { return header_; }
};
} // namespace movilcam
