// SolarisPKN MovilCam Windows Broker - console prototype.
// 1 ADB TCP connection; 1 decoder; 3 NV12 shared memory slots for camera consumers.
#define WIN32_LEAN_AND_MEAN
#define NOMINMAX
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#include <mfapi.h>
#include <cstdio>
#include <cstdint>
#include <atomic>
#include <vector>
#include <string>
#include <algorithm>
#include "protocol.hpp"
#include "decoder.hpp"
#include "ring.hpp"

#pragma comment(lib,"ws2_32.lib")
#pragma comment(lib,"mfplat.lib")
#pragma comment(lib,"mfuuid.lib")
#pragma comment(lib,"ole32.lib")

namespace {
std::atomic<bool> run{true};
BOOL WINAPI control(DWORD kind) {
    if (kind == CTRL_C_EVENT || kind == CTRL_BREAK_EVENT ||
        kind == CTRL_CLOSE_EVENT || kind == CTRL_SHUTDOWN_EVENT) {
        run.store(false); return TRUE;
    }
    return FALSE;
}
struct Socket {
    SOCKET handle = INVALID_SOCKET;
    explicit Socket(SOCKET h = INVALID_SOCKET) : handle(h) {}
    ~Socket() { if(handle!=INVALID_SOCKET) closesocket(handle); }
    Socket(const Socket&) = delete;
    Socket& operator=(const Socket&) = delete;
};
bool recv_all(SOCKET socket, std::uint8_t* dst, std::size_t bytes) {
    while (bytes && run) {
        int n = recv(socket,reinterpret_cast<char*>(dst),
                     static_cast<int>(std::min<std::size_t>(bytes,64*1024)),0);
        if (n <= 0) return false;
        dst += n; bytes -= n;
    }
    return bytes == 0;
}
bool send_all(SOCKET socket, const std::uint8_t* data, std::size_t bytes) {
    while (bytes && run) {
        int n=send(socket,reinterpret_cast<const char*>(data),
                   static_cast<int>(bytes),0);
        if(n <= 0) return false;
        data+=n; bytes-=n;
    }
    return bytes == 0;
}
bool once(bool d3d) {
    Socket sock(socket(AF_INET,SOCK_STREAM,IPPROTO_TCP));
    if(sock.handle==INVALID_SOCKET) return false;
    DWORD timeout=3000;
    setsockopt(sock.handle,SOL_SOCKET,SO_RCVTIMEO,
               reinterpret_cast<const char*>(&timeout),sizeof(timeout));
    setsockopt(sock.handle,SOL_SOCKET,SO_SNDTIMEO,
               reinterpret_cast<const char*>(&timeout),sizeof(timeout));
    int noDelay=1;
    setsockopt(sock.handle,IPPROTO_TCP,TCP_NODELAY,
               reinterpret_cast<const char*>(&noDelay),sizeof(noDelay));
    sockaddr_in addr{};
    addr.sin_family=AF_INET; addr.sin_port=htons(27183);
    InetPtonA(AF_INET,"127.0.0.1",&addr.sin_addr);
    if(connect(sock.handle,reinterpret_cast<sockaddr*>(&addr),sizeof(addr))!=0)
        return false;
    if(!send_all(sock.handle,movilcam::hello.data(),movilcam::hello.size()))
        return false;
    std::uint8_t hdr[movilcam::stream_header_bytes]{};
    if(!recv_all(sock.handle,hdr,sizeof(hdr))) return false;
    movilcam::StreamHeader h{};
    if(!movilcam::parse_stream(hdr,sizeof(hdr),h)) {
        std::fprintf(stderr,"[MovilCam] Android app old version or protocol mismatch.\n");
        return false;
    }
    movilcam::FrameRing ring;
    if(!ring.create(h.width,h.height)) {
        std::fprintf(stderr,"[MovilCam] NV12 ring already exists or memory failure. One broker only.\n");
        return false;
    }
    movilcam::H264Decoder decoder;
    if(!decoder.open(h.width,h.height,h.fps,d3d)) {
        std::fprintf(stderr,"[MovilCam] Media Foundation H264 decoder unavailable.\n");
        return false;
    }
    std::printf("[MovilCam] %ux%u @ %u FPS, D3D11 requested=%s, active=%s\n",
                h.width,h.height,h.fps,d3d?"yes":"no",
                decoder.gpu_enabled()?"yes":"no");
    std::printf("[MovilCam] NV12 triple-buffer: Local\\SolarisPKN_MovilCam_NV12_v1.\n");
    std::vector<std::uint8_t> payload;
    std::uint64_t packets=0, failures=0;
    while(run) {
        std::uint8_t raw[movilcam::packet_header_bytes];
        if(!recv_all(sock.handle,raw,sizeof(raw))) break;
        movilcam::PacketHeader p{};
        if(!movilcam::parse_packet(raw,sizeof(raw),p)) {
            std::fprintf(stderr,"[MovilCam] Bad or oversized H264 frame. Disconnecting.\n");
            break;
        }
        payload.resize(p.size);
        if(!recv_all(sock.handle,payload.data(),p.size)) break;
        // Media Foundation input timestamp is expressed in 100-nanosecond units.
        if(!decoder.decode(payload.data(),payload.size(),
                          static_cast<LONGLONG>(p.pts_us)*10,ring)) {
            ++failures;
            if(failures>12) {
                std::fprintf(stderr,"[MovilCam] Decoder failing repeatedly; reconnect.\n");
                break;
            }
        } else failures=0;
        ++packets;
        if(packets%300==0) {
            auto* state=ring.header();
            std::printf("[MovilCam] H264 access units=%llu, decoded frames=%lld\n",
                        static_cast<unsigned long long>(packets),
                        static_cast<long long>(state->totalFrames));
        }
    }
    return true;
}
} // namespace

int main(int argc,char** argv) {
    bool hardware=true;
    for(int i=1;i<argc;++i) {
        if(std::string(argv[i])=="--software") hardware=false;
        else if(std::string(argv[i])=="--help") {
            std::puts("MovilCamBridge [--software]\n"
                "Requires adb forward tcp:27183 tcp:27183 and MovilCam Android running.\n"
                "Connects once; decodes H264 to NV12; publishes triple-buffer for future DShow driver.");
            return 0;
        }
    }
    SetConsoleCtrlHandler(control,TRUE);
    WSADATA ws{};
    if(WSAStartup(MAKEWORD(2,2),&ws)!=0) return 1;
    HRESULT com=CoInitializeEx(nullptr,COINIT_MULTITHREADED);
    if(FAILED(com)){ WSACleanup();return 2; }
    HRESULT mf=MFStartup(MF_VERSION,MFSTARTUP_FULL);
    if(FAILED(mf)){CoUninitialize();WSACleanup();return 3;}
    while(run) {
        if(!once(hardware) && run) {
            std::fprintf(stderr,"[MovilCam] Waiting for ADB camera...\n");
        }
        for(int x=0;x<10&&run;++x) Sleep(100);
    }
    MFShutdown(); CoUninitialize(); WSACleanup();
    return 0;
}
