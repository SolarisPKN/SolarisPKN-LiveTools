#include "control.hpp"
#include <cstdio>
namespace movilcam {
bool ControlServer::start() {
    if(acceptor_.joinable()) return false;
    state_->running.store(true);
    auto s=state_;
    acceptor_=std::thread([s]{
        while(s->running.load()) {
            HANDLE pipe=CreateNamedPipeW(control_pipe,
                PIPE_ACCESS_INBOUND,
                PIPE_TYPE_BYTE|PIPE_READMODE_BYTE|PIPE_WAIT|PIPE_REJECT_REMOTE_CLIENTS,
                PIPE_UNLIMITED_INSTANCES,64,64,0,nullptr);
            if(pipe==INVALID_HANDLE_VALUE) {
                std::fprintf(stderr,"[MovilCam] Control pipe creation failed: %lu\n",GetLastError());
                break;
            }
            BOOL connected=ConnectNamedPipe(pipe,nullptr) ?
                TRUE : GetLastError()==ERROR_PIPE_CONNECTED;
            if(!connected || !s->running.load()) {CloseHandle(pipe);continue;}
            std::thread([s,pipe]{
                s->clients.fetch_add(1);
                std::fprintf(stderr,"[MovilCam] Camera users: %d\n",s->clients.load());
                char data[16];DWORD bytes=0;
                // A writer's disconnect unblocks ReadFile even if it never sends data.
                while(ReadFile(pipe,data,sizeof(data),&bytes,nullptr)) {
                    if(bytes==0) break;
                }
                s->clients.fetch_sub(1);
                std::fprintf(stderr,"[MovilCam] Camera users: %d\n",s->clients.load());
                DisconnectNamedPipe(pipe);
                CloseHandle(pipe);
            }).detach();
        }
    });
    return true;
}
void ControlServer::stop(){
    if(!acceptor_.joinable()) return;
    state_->running.store(false);
    // Wake ConnectNamedPipe, allowing the acceptor thread to exit.
    HANDLE poke=CreateFileW(control_pipe,GENERIC_WRITE,0,nullptr,OPEN_EXISTING,0,nullptr);
    if(poke!=INVALID_HANDLE_VALUE) CloseHandle(poke);
    acceptor_.join();
}
}
