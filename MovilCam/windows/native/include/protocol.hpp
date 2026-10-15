#pragma once
// SolarisPKN MovilCam protocol v2. Network-byte-order headers, no malloc parser.
// Handshake sent by Windows client to Android: "MCAMV002" (8 bytes).
// Android responds with 20-byte stream header and length-delimited access units.
#include <array>
#include <cstddef>
#include <cstdint>

namespace movilcam {
inline constexpr std::array<std::uint8_t, 8> hello = {'M','C','A','M','V','0','0','2'};
inline constexpr std::size_t stream_header_bytes = 20;
inline constexpr std::size_t packet_header_bytes = 16;
inline constexpr std::uint32_t max_payload = 2 * 1024 * 1024;
inline constexpr std::uint32_t flag_keyframe = 1;
inline constexpr std::uint32_t flag_config = 2;

inline std::uint32_t be32(const std::uint8_t* p) noexcept {
    return (std::uint32_t(p[0]) << 24) | (std::uint32_t(p[1]) << 16) |
           (std::uint32_t(p[2]) << 8) | p[3];
}
inline std::uint64_t be64(const std::uint8_t* p) noexcept {
    return (std::uint64_t(be32(p)) << 32) | be32(p + 4);
}
inline void put32(std::uint8_t* p, std::uint32_t x) noexcept {
    p[0] = std::uint8_t(x >> 24);
    p[1] = std::uint8_t(x >> 16);
    p[2] = std::uint8_t(x >> 8);
    p[3] = std::uint8_t(x);
}
inline void put64(std::uint8_t* p, std::uint64_t x) noexcept {
    put32(p, std::uint32_t(x >> 32)); put32(p + 4, std::uint32_t(x));
}
struct StreamHeader { std::uint32_t width{}, height{}, fps{}, flags{}; };
struct PacketHeader { std::uint32_t size{}, flags{}; std::uint64_t pts_us{}; };
inline bool parse_stream(const std::uint8_t* p, std::size_t n, StreamHeader& s) noexcept {
    if (n != stream_header_bytes || p[0] != 'M' || p[1] != 'C' ||
        p[2] != 'V' || p[3] != '2') return false;
    s = {be32(p+4), be32(p+8), be32(p+12), be32(p+16)};
    return s.width >= 160 && s.width <= 3840 && s.height >= 120 &&
           s.height <= 2160 && s.fps >= 1 && s.fps <= 120 &&
           (s.width % 2) == 0 && (s.height % 2) == 0;
}
inline bool parse_packet(const std::uint8_t* p, std::size_t n, PacketHeader& h) noexcept {
    if (n != packet_header_bytes) return false;
    h = {be32(p), be32(p+4), be64(p+8)};
    return h.size > 0 && h.size <= max_payload && (h.flags & ~3u) == 0;
}
} // namespace movilcam
