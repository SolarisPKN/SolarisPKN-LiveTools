#include "protocol.hpp"
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <cassert>
#include <array>
int main() {
    using namespace movilcam;
    std::array<std::uint8_t,20> hdr = {'M','C','V','2'};
    put32(hdr.data()+4,1920); put32(hdr.data()+8,1080);
    put32(hdr.data()+12,30); put32(hdr.data()+16,0);
    StreamHeader s;
    assert(parse_stream(hdr.data(),hdr.size(),s));
    assert(s.width==1920 && s.height==1080 && s.fps==30);
    hdr[0]='X'; assert(!parse_stream(hdr.data(),hdr.size(),s));
    std::array<std::uint8_t,16> p{};
    put32(p.data(),1024); put32(p.data()+4,flag_keyframe);
    put64(p.data()+8,0x000001230000abcdULL);
    PacketHeader packet;
    assert(parse_packet(p.data(),p.size(),packet));
    assert(packet.size==1024 && packet.flags==flag_keyframe);
    assert(packet.pts_us==0x000001230000abcdULL);
    put32(p.data(),max_payload+1);
    assert(!parse_packet(p.data(),p.size(),packet));
    return 0;
}
