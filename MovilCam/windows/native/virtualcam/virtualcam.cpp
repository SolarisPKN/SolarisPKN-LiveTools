// SolarisPKN MovilCam DirectShow source (Windows 10 LTSC compatibility).
// Reads the broker's shared NV12 triple buffer; never opens ADB or decodes video.
// Preferred NV12 path copies exactly once into the receiving app's sample;
// optional YUY2 conversion is enabled ONLY when consumer requests it.
#include <streams.h>
#include <initguid.h>
#include <ks.h>
#include <ksmedia.h>
#include <algorithm>
#include <cstdint>
#include <cstring>
#include <vector>
#include "ring.hpp"

DEFINE_GUID(CLSID_SolarisPKNMovilCam,0x4b34118d,0x1f57,0x41ef,0xa8,0x22,0x27,0x6b,0x9c,0x11,0xca,0x01);
DEFINE_GUID(MOVILCAM_NV12,0x3231564e,0x0000,0x0010,0x80,0x00,0x00,0xaa,0x00,0x38,0x9b,0x71);
DEFINE_GUID(MOVILCAM_YUY2,0x32595559,0x0000,0x0010,0x80,0x00,0x00,0xaa,0x00,0x38,0x9b,0x71);

static const wchar_t camera_name[] = L"SolarisPKN MovilCam";
static constexpr unsigned kMaxDim=3840;

class SharedReader {
    HANDLE handle_ = nullptr;
    const std::uint8_t* data_ = nullptr;
    const movilcam::RingHeader* hdr_ = nullptr;
    DWORD width_ = 1920, height_ = 1080;
    LONGLONG lastFrame_ = 0;
public:
    ~SharedReader(){ close(); }
    void close() {
        hdr_=nullptr;
        if(data_) UnmapViewOfFile(data_);
        data_=nullptr;
        if(handle_) CloseHandle(handle_);
        handle_=nullptr;
    }
    bool open() {
        if(hdr_) return true;
        handle_=OpenFileMappingW(FILE_MAP_READ,FALSE,movilcam::ring_name);
        if(!handle_) return false;
        data_=static_cast<const std::uint8_t*>(MapViewOfFile(handle_,FILE_MAP_READ,0,0,0));
        if(!data_){close();return false;}
        hdr_=reinterpret_cast<const movilcam::RingHeader*>(data_);
        if(hdr_->magic != movilcam::ring_magic || hdr_->version != movilcam::ring_version ||
           hdr_->width < 160 || hdr_->width > kMaxDim || hdr_->height < 120 ||
           hdr_->height > 2160 || (hdr_->width & 1) || (hdr_->height & 1) ||
           hdr_->pitch != hdr_->width ||
           hdr_->frameBytes != std::uint64_t(hdr_->width)*hdr_->height*3/2 ||
           hdr_->slotStride < hdr_->frameBytes+sizeof(movilcam::SlotHeader) ||
           hdr_->slotStride > 16*1024*1024) {close();return false;}
        width_=hdr_->width; height_=hdr_->height;
        return true;
    }
    DWORD width() const {return width_;}
    DWORD height() const {return height_;}
    LONGLONG latest() const {return hdr_ ? hdr_->published : 0;}
    bool copy_latest(std::uint8_t* dst, size_t capacity, LONGLONG& seq) {
        if(!hdr_ || capacity < hdr_->frameBytes) return false;
        for(int i=0;i<3;++i){
            MemoryBarrier();
            const LONGLONG published=hdr_->published;
            if(published<=0) return false;
            const auto index=(published-1)%movilcam::ring_slots;
            const auto* slot=reinterpret_cast<const movilcam::SlotHeader*>(
                data_+sizeof(movilcam::RingHeader)+index*hdr_->slotStride);
            const LONGLONG expected=published*2;
            if(slot->sequence != expected || slot->bytes != hdr_->frameBytes) continue;
            MemoryBarrier();
            std::memcpy(dst,reinterpret_cast<const std::uint8_t*>(slot)+
                        sizeof(movilcam::SlotHeader),hdr_->frameBytes);
            MemoryBarrier();
            if(slot->sequence==expected){
                seq=published;
                lastFrame_=published;
                return true;
            }
        }
        return false;
    }
};

static std::uint8_t clip(int x) {
    return static_cast<std::uint8_t>(std::min(255,std::max(0,x)));
}
// NV12 -> YUY2, 2 pixels per chroma pair. No RGB math, no intermediate frame copies.
static void nv12_to_yuy2(const std::uint8_t* src,std::uint8_t* dst,
                         unsigned width,unsigned height) {
    const auto* uv=src+size_t(width)*height;
    for(unsigned y=0;y<height;++y){
        const auto* yrow=src+size_t(y)*width;
        const auto* crow=uv+size_t(y/2)*width;
        auto* row=dst+size_t(y)*width*2;
        for(unsigned x=0;x<width;x+=2){
            row[2*x+0]=yrow[x];
            row[2*x+1]=crow[x];
            row[2*x+2]=yrow[x+1];
            row[2*x+3]=crow[x+1];
        }
    }
}

class CameraPin final : public CSourceStream, public IAMStreamConfig, public IKsPropertySet {
    SharedReader reader_;
    DWORD width_=1920,height_=1080,fps_=30;
    REFERENCE_TIME frameTime_=333333;
    REFERENCE_TIME sampleTime_=0;
    LONGLONG lastSeen_=0;
    bool useYUY2_=false;
    std::vector<std::uint8_t> scratch_;
    DWORD reopenCounter_=0;
public:
    CameraPin(HRESULT* hr,CSource* parent):
        CSourceStream(NAME("SolarisPKN MovilCam output"),hr,parent,L"Capture"){
        if(reader_.open()){width_=reader_.width();height_=reader_.height();}
    }
    STDMETHODIMP NonDelegatingQueryInterface(REFIID riid,void** ppv) override {
        if(riid==IID_IAMStreamConfig) return GetInterface(static_cast<IAMStreamConfig*>(this),ppv);
        if(riid==IID_IKsPropertySet) return GetInterface(static_cast<IKsPropertySet*>(this),ppv);
        return CSourceStream::NonDelegatingQueryInterface(riid,ppv);
    }
    HRESULT GetMediaType(int position,CMediaType* mt) override {
        if(!mt) return E_POINTER;
        if(position<0) return E_INVALIDARG;
        if(position>1) return VFW_S_NO_MORE_ITEMS;
        const bool yuy2=position==1;
        mt->InitMediaType();
        auto* vih=reinterpret_cast<VIDEOINFOHEADER*>(
            mt->AllocFormatBuffer(sizeof(VIDEOINFOHEADER)));
        if(!vih) return E_OUTOFMEMORY;
        ZeroMemory(vih,sizeof(VIDEOINFOHEADER));
        vih->bmiHeader.biSize=sizeof(BITMAPINFOHEADER);
        vih->bmiHeader.biWidth=LONG(width_);
        vih->bmiHeader.biHeight=LONG(height_);
        vih->bmiHeader.biPlanes=1;
        vih->bmiHeader.biBitCount=WORD(yuy2?16:12);
        vih->bmiHeader.biCompression=yuy2?MAKEFOURCC('Y','U','Y','2'):MAKEFOURCC('N','V','1','2');
        vih->bmiHeader.biSizeImage=width_*height_*(yuy2?2:3)/ (yuy2?1:2);
        vih->AvgTimePerFrame=frameTime_;
        mt->SetType(&MEDIATYPE_Video);
        mt->SetSubtype(yuy2?&MOVILCAM_YUY2:&MOVILCAM_NV12);
        mt->SetFormatType(&FORMAT_VideoInfo);
        mt->SetTemporalCompression(FALSE);
        mt->SetSampleSize(vih->bmiHeader.biSizeImage);
        return S_OK;
    }
    HRESULT CheckMediaType(const CMediaType* mt) override {
        if(!mt || *mt->Type()!=MEDIATYPE_Video ||
           *mt->FormatType()!=FORMAT_VideoInfo || !mt->IsFixedSize()) return VFW_E_TYPE_NOT_ACCEPTED;
        const bool isNV12=*mt->Subtype()==MOVILCAM_NV12;
        const bool isYUY2=*mt->Subtype()==MOVILCAM_YUY2;
        if(!isNV12&&!isYUY2) return VFW_E_TYPE_NOT_ACCEPTED;
        if(mt->FormatLength()<sizeof(VIDEOINFOHEADER)) return VFW_E_TYPE_NOT_ACCEPTED;
        auto* vih=reinterpret_cast<const VIDEOINFOHEADER*>(mt->Format());
        if(vih->bmiHeader.biWidth!=LONG(width_) || vih->bmiHeader.biHeight!=LONG(height_))
            return VFW_E_TYPE_NOT_ACCEPTED;
        return S_OK;
    }
    HRESULT SetMediaType(const CMediaType* type) override {
        const HRESULT result=CSourceStream::SetMediaType(type);
        if(SUCCEEDED(result)) useYUY2_=(*type->Subtype()==MOVILCAM_YUY2);
        return result;
    }
    HRESULT DecideBufferSize(IMemAllocator* allocator,ALLOCATOR_PROPERTIES* props) override {
        if(!allocator||!props) return E_POINTER;
        const DWORD frameBytes=width_*height_*(useYUY2_?2:3)/(useYUY2_?1:2);
        props->cBuffers=2;
        props->cbBuffer=LONG(frameBytes);
        props->cbAlign=1; props->cbPrefix=0;
        ALLOCATOR_PROPERTIES actual{};
        HRESULT hr=allocator->SetProperties(props,&actual);
        if(FAILED(hr)) return hr;
        return actual.cbBuffer>=LONG(frameBytes)?S_OK:E_FAIL;
    }
    HRESULT OnThreadCreate() override {
        sampleTime_=0;lastSeen_=0;
        scratch_.resize(size_t(width_)*height_*3/2);
        return S_OK;
    }
    HRESULT OnThreadDestroy() override {
        scratch_.clear();return S_OK;
    }
    HRESULT FillBuffer(IMediaSample* sample) override {
        if(!sample)return E_POINTER;
        BYTE* dst=nullptr;
        if(FAILED(sample->GetPointer(&dst))) return E_FAIL;
        const size_t nv12size=size_t(width_)*height_*3/2;
        const size_t outsize=useYUY2_?size_t(width_)*height_*2:nv12size;
        if(sample->GetSize()<LONG(outsize)) return E_FAIL;
        // Wait for a fresh frame, but never stall forever if USB drops.
        if(!reader_.open()) {
            if((++reopenCounter_%40)==0) reader_.close();
        }
        LONGLONG seen=reader_.latest();
        for(int wait=0;wait<35 && seen==lastSeen_;++wait){
            Sleep(2);
            seen=reader_.latest();
        }
        LONGLONG copied=0;
        bool got=false;
        if(reader_.width()==width_ && reader_.height()==height_) {
            if(useYUY2_) {
                got=reader_.copy_latest(scratch_.data(),scratch_.size(),copied);
                if(got) nv12_to_yuy2(scratch_.data(),dst,width_,height_);
            } else {
                got=reader_.copy_latest(dst,nv12size,copied);
            }
        }
        if(got) lastSeen_=copied;
        else {
            // Always output deterministic black when no valid frame exists.
            // Never deliver uninitialized sample memory to apps after a disconnect.
            if(!useYUY2_) {
                std::memset(dst,16,size_t(width_)*height_);
                std::memset(dst+size_t(width_)*height_,128,nv12size-width_*height_);
            } else {
                for(size_t i=0;i<outsize;i+=4) {dst[i]=16;dst[i+1]=128;dst[i+2]=16;dst[i+3]=128;}
            }
        }
        sample->SetActualDataLength(LONG(outsize));
        REFERENCE_TIME start=sampleTime_,end=start+frameTime_;
        sample->SetTime(&start,&end);
        sample->SetSyncPoint(TRUE);
        sampleTime_=end;
        return S_OK;
    }
    // IAMStreamConfig allows apps to enumerate the camera formats.
    STDMETHODIMP SetFormat(AM_MEDIA_TYPE* mt) override {
        CMediaType requested(*mt);
        return CheckMediaType(&requested);
    }
    STDMETHODIMP GetFormat(AM_MEDIA_TYPE** mt) override {
        if(!mt) return E_POINTER;
        CMediaType proposed;
        HRESULT hr=GetMediaType(useYUY2_?1:0,&proposed);
        if(FAILED(hr)) return hr;
        *mt=CreateMediaType(&proposed);
        return *mt?S_OK:E_OUTOFMEMORY;
    }
    STDMETHODIMP GetNumberOfCapabilities(int* count,int* size) override {
        if(!count||!size)return E_POINTER;
        *count=2; *size=sizeof(VIDEO_STREAM_CONFIG_CAPS);
        return S_OK;
    }
    STDMETHODIMP GetStreamCaps(int index,AM_MEDIA_TYPE** mt,BYTE* data) override {
        if(!mt||!data)return E_POINTER;
        CMediaType proposed;
        HRESULT hr=GetMediaType(index,&proposed);
        if(FAILED(hr))return hr;
        *mt=CreateMediaType(&proposed);
        if(!*mt)return E_OUTOFMEMORY;
        auto* caps=reinterpret_cast<VIDEO_STREAM_CONFIG_CAPS*>(data);
        ZeroMemory(caps,sizeof(*caps));
        caps->guid=FORMAT_VideoInfo;
        caps->InputSize={LONG(width_),LONG(height_)};
        caps->MinOutputSize=caps->MaxOutputSize=caps->InputSize;
        caps->MinFrameInterval=caps->MaxFrameInterval=frameTime_;
        caps->MinBitsPerSecond=LONG(width_*height_*12*fps_);
        caps->MaxBitsPerSecond=LONG(width_*height_*16*fps_);
        return S_OK;
    }
    // IKsPropertySet is used by a number of Windows apps to identify a capture pin.
    STDMETHODIMP Set(REFGUID,DWORD,LPVOID,DWORD,LPVOID,DWORD) override {
        return E_NOTIMPL;
    }
    STDMETHODIMP Get(REFGUID propSet,DWORD propId,LPVOID,DWORD,
                     LPVOID out,DWORD capacity,DWORD* returned) override {
        if(propSet!=AMPROPSETID_Pin || propId!=AMPROPERTY_PIN_CATEGORY)
            return E_PROP_ID_UNSUPPORTED;
        if(returned) *returned=sizeof(GUID);
        if(!out) return E_POINTER;
        if(capacity<sizeof(GUID))return E_UNEXPECTED;
        *reinterpret_cast<GUID*>(out)=PIN_CATEGORY_CAPTURE;
        return S_OK;
    }
    STDMETHODIMP QuerySupported(REFGUID propSet,DWORD propId,DWORD* support) override {
        if(!support) return E_POINTER;
        if(propSet!=AMPROPSETID_Pin || propId!=AMPROPERTY_PIN_CATEGORY)
            return E_PROP_ID_UNSUPPORTED;
        *support=KSPROPERTY_SUPPORT_GET;return S_OK;
    }
};

class CameraFilter final: public CSource {
public:
    CameraFilter(LPUNKNOWN unk,HRESULT* hr):
        CSource(NAME("SolarisPKN MovilCam"),unk,CLSID_SolarisPKNMovilCam) {
        new CameraPin(hr,this);
    }
    static CUnknown* WINAPI CreateInstance(LPUNKNOWN unk,HRESULT* hr) {
        auto* filter=new CameraFilter(unk,hr);
        if(!filter && hr) *hr=E_OUTOFMEMORY;
        return filter;
    }
};

const AMOVIESETUP_MEDIATYPE setupMedia = { &MEDIATYPE_Video,&MOVILCAM_NV12 };
const AMOVIESETUP_PIN setupPin = {
    L"Capture",FALSE,TRUE,FALSE,FALSE,&CLSID_NULL,nullptr,1,&setupMedia
};
const AMOVIESETUP_FILTER setupFilter = {
    &CLSID_SolarisPKNMovilCam,camera_name,MERIT_DO_NOT_USE,1,&setupPin
};
CFactoryTemplate g_Templates[] = {
    {camera_name,&CLSID_SolarisPKNMovilCam,CameraFilter::CreateInstance,nullptr,&setupFilter}
};
int g_cTemplates=sizeof(g_Templates)/sizeof(g_Templates[0]);

STDAPI DllRegisterServer() {
    HRESULT hr=AMovieDllRegisterServer2(TRUE);
    if(FAILED(hr))return hr;
    IFilterMapper2* mapper=nullptr;
    hr=CoCreateInstance(CLSID_FilterMapper2,nullptr,CLSCTX_INPROC_SERVER,
                        IID_IFilterMapper2,reinterpret_cast<void**>(&mapper));
    if(FAILED(hr))return hr;
    const REGPINTYPES type = { &MEDIATYPE_Video,&MOVILCAM_NV12 };
    const REGFILTERPINS2 pin = { REG_PINFLAG_B_OUTPUT,1,1,&type,0,nullptr,&PIN_CATEGORY_CAPTURE };
    REGFILTER2 filter{};
    filter.dwVersion=2;
    filter.dwMerit=MERIT_DO_NOT_USE;
    filter.cPins2=1;
    filter.rgPins2=&pin;
    hr=mapper->RegisterFilter(CLSID_SolarisPKNMovilCam,camera_name,nullptr,
                              &CLSID_VideoInputDeviceCategory,camera_name,&filter);
    mapper->Release();
    return hr;
}
STDAPI DllUnregisterServer() {
    IFilterMapper2* mapper=nullptr;
    if(SUCCEEDED(CoCreateInstance(CLSID_FilterMapper2,nullptr,CLSCTX_INPROC_SERVER,
                    IID_IFilterMapper2,reinterpret_cast<void**>(&mapper)))) {
        mapper->UnregisterFilter(&CLSID_VideoInputDeviceCategory,camera_name,
                                 CLSID_SolarisPKNMovilCam);
        mapper->Release();
    }
    return AMovieDllRegisterServer2(FALSE);
}
extern "C" BOOL WINAPI DllEntryPoint(HINSTANCE,ULONG,LPVOID);
BOOL APIENTRY DllMain(HANDLE module,DWORD reason,LPVOID context) {
    return DllEntryPoint(reinterpret_cast<HINSTANCE>(module),reason,context);
}
