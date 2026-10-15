#include "ring.hpp"
#include <cstring>
#include <limits>

namespace movilcam {
bool FrameRing::create(DWORD width, DWORD height) {
    close();
    if (!width || !height || width > 3840 || height > 2160 ||
        (width & 1) || (height & 1)) return false;
    const std::uint64_t payload = std::uint64_t(width) * height * 3 / 2;
    const std::uint64_t stride = ((sizeof(SlotHeader) + payload + 63) / 64) * 64;
    const std::uint64_t size = sizeof(RingHeader) + stride * ring_slots;
    if (size > std::numeric_limits<DWORD>::max()) return false;
    mapping_ = CreateFileMappingW(INVALID_HANDLE_VALUE, nullptr, PAGE_READWRITE,
                                  DWORD(size >> 32), DWORD(size), ring_name);
    if (!mapping_) return false;
    // Never share an unrelated/stale mapping with an unknown layout.
    if (GetLastError() == ERROR_ALREADY_EXISTS) { close(); return false; }
    base_ = static_cast<std::uint8_t*>(MapViewOfFile(mapping_,FILE_MAP_WRITE | FILE_MAP_READ,0,0,size));
    if (!base_) { close(); return false; }
    std::memset(base_,0,static_cast<std::size_t>(size));
    stride_ = static_cast<DWORD>(stride);
    header_ = reinterpret_cast<RingHeader*>(base_);
    header_->width = width; header_->height = height;
    header_->pitch = width; header_->frameBytes = static_cast<DWORD>(payload);
    header_->slotStride = stride_;
    // Publish magic last, to avoid consumers reading incomplete header.
    MemoryBarrier();
    header_->magic = ring_magic;
    next_ = 0;
    return true;
}
void FrameRing::close() noexcept {
    header_ = nullptr;
    if (base_) UnmapViewOfFile(base_);
    base_ = nullptr;
    if (mapping_) CloseHandle(mapping_);
    mapping_ = nullptr;
    stride_ = 0;
}
bool FrameRing::publish(const std::uint8_t* nv12, std::size_t bytes,
                        LONGLONG timestamp100ns) noexcept {
    if (!header_ || !nv12 || bytes != header_->frameBytes) return false;
    const std::uint64_t index = next_++ % ring_slots;
    auto* slot = reinterpret_cast<SlotHeader*>(
        base_ + sizeof(RingHeader) + index * stride_);
    // Dedicated writer; seq is odd during the copy and even only after it completes.
    InterlockedExchange64(&slot->sequence, LONG64(next_ * 2 - 1));
    MemoryBarrier();
    std::memcpy(reinterpret_cast<std::uint8_t*>(slot) + sizeof(SlotHeader),
                nv12, bytes);
    slot->timestamp100ns = timestamp100ns;
    slot->bytes = static_cast<DWORD>(bytes);
    MemoryBarrier();
    InterlockedExchange64(&slot->sequence, LONG64(next_ * 2));
    InterlockedExchange64(&header_->published, LONG64(next_));
    InterlockedIncrement64(&header_->totalFrames);
    LARGE_INTEGER qpc{};
    QueryPerformanceCounter(&qpc);
    InterlockedExchange64(&header_->lastQpc, qpc.QuadPart);
    return true;
}
} // namespace movilcam
