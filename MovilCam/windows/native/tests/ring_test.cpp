#include "ring.hpp"
#ifdef NDEBUG
#undef NDEBUG
#endif
#include <cassert>
#include <vector>
#include <cstring>
#include <algorithm>
int main(){
    using namespace movilcam;
    FrameRing ring;
    assert(ring.create(320,240,60));
    std::vector<std::uint8_t> data(320*240*3/2,128);
    std::fill(data.begin(),data.begin()+320*240,16);
    assert(ring.publish(data.data(),data.size(),123456));
    HANDLE map=OpenFileMappingW(FILE_MAP_READ,FALSE,ring_name);
    assert(map!=nullptr);
    const auto* base=static_cast<const std::uint8_t*>(
        MapViewOfFile(map,FILE_MAP_READ,0,0,0));
    assert(base!=nullptr);
    const auto* h=reinterpret_cast<const RingHeader*>(base);
    assert(h->magic==ring_magic && h->version==ring_version);
    assert(h->published==1 && h->totalFrames==1);
    assert(h->frameBytes==data.size());
    assert(h->fps==60);
    auto* slot=reinterpret_cast<const SlotHeader*>(base+sizeof(RingHeader));
    assert(slot->sequence==2 && slot->timestamp100ns==123456);
    const auto* bytes=reinterpret_cast<const std::uint8_t*>(slot)+sizeof(SlotHeader);
    assert(std::memcmp(bytes,data.data(),data.size())==0);
    assert(ring.publish(data.data(),data.size(),222222));
    assert(h->published==2 && h->totalFrames==2);
    ring.close();
    // Windows keeps mapped objects alive while a client holds a reference.
    // Broker must not recreate the ring while any consumer still has it open.
    assert(!ring.create(320,240,60));
    UnmapViewOfFile(base);
    CloseHandle(map);
    assert(ring.create(320,240,60));
    ring.close();
    return 0;
}
