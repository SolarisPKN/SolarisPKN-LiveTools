// Integration smoke test: Windows must enumerate and instantiate MovilCam as
// a video capture source after installing the DLL. No Android phone required.
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <dshow.h>
#include <cstdio>
#include <cwchar>
#pragma comment(lib,"strmiids.lib")
#pragma comment(lib,"ole32.lib")
#pragma comment(lib,"oleaut32.lib")

static int test() {
    ICreateDevEnum* systemEnum=nullptr;
    HRESULT hr=CoCreateInstance(CLSID_SystemDeviceEnum,nullptr,CLSCTX_INPROC_SERVER,
                                IID_ICreateDevEnum,reinterpret_cast<void**>(&systemEnum));
    if(FAILED(hr)){std::fprintf(stderr,"SystemDeviceEnum unavailable: 0x%08lX\n",hr);return 2;}
    IEnumMoniker* enumerator=nullptr;
    hr=systemEnum->CreateClassEnumerator(CLSID_VideoInputDeviceCategory,&enumerator,0);
    systemEnum->Release();
    if(hr!=S_OK || !enumerator){
        std::fprintf(stderr,"No video input devices registered\n");return 3;
    }
    int result=4;
    IMoniker* moniker=nullptr;
    while(enumerator->Next(1,&moniker,nullptr)==S_OK){
        IPropertyBag* bag=nullptr;
        if(SUCCEEDED(moniker->BindToStorage(nullptr,nullptr,IID_IPropertyBag,
                         reinterpret_cast<void**>(&bag)))) {
            VARIANT name; VariantInit(&name);
            if(SUCCEEDED(bag->Read(L"FriendlyName",&name,nullptr)) &&
                    name.vt==VT_BSTR && wcscmp(name.bstrVal,L"SolarisPKN MovilCam")==0) {
                IBaseFilter* filter=nullptr;
                hr=moniker->BindToObject(nullptr,nullptr,IID_IBaseFilter,
                                        reinterpret_cast<void**>(&filter));
                if(FAILED(hr)){
                    std::fprintf(stderr,"Camera registered but cannot instantiate: 0x%08lX\n",hr);
                    result=5;
                }else{
                    IEnumPins* pins=nullptr;
                    if(FAILED(filter->EnumPins(&pins)) || !pins){
                        std::fprintf(stderr,"Camera instantiated without pin enumerator\n");
                        result=6;
                    }else{
                        IPin* pin=nullptr;
                        result=7;
                        while(pins->Next(1,&pin,nullptr)==S_OK){
                            PIN_DIRECTION direction=PINDIR_INPUT;
                            pin->QueryDirection(&direction);
                            IAMStreamConfig* cfg=nullptr;
                            if(direction==PINDIR_OUTPUT &&
                               SUCCEEDED(pin->QueryInterface(IID_IAMStreamConfig,
                                             reinterpret_cast<void**>(&cfg)))) {
                                int count=0,size=0;
                                hr=cfg->GetNumberOfCapabilities(&count,&size);
                                if(SUCCEEDED(hr) && count>=2 && size==sizeof(VIDEO_STREAM_CONFIG_CAPS)){
                                    AM_MEDIA_TYPE* media=nullptr;
                                    VIDEO_STREAM_CONFIG_CAPS caps{};
                                    hr=cfg->GetStreamCaps(0,&media,reinterpret_cast<BYTE*>(&caps));
                                    if(SUCCEEDED(hr) && media && media->cbFormat>=sizeof(VIDEOINFOHEADER) &&
                                        media->pbFormat) {
                                        auto* vih=reinterpret_cast<const VIDEOINFOHEADER*>(media->pbFormat);
                                        if(vih->bmiHeader.biWidth==1920 && vih->bmiHeader.biHeight==1080 &&
                                            vih->AvgTimePerFrame>=166666 && vih->AvgTimePerFrame<=166668 &&
                                            caps.MinFrameInterval==vih->AvgTimePerFrame &&
                                            caps.MaxFrameInterval==vih->AvgTimePerFrame) {
                                            std::printf("MovilCam enumerated: 1920x1080 60 FPS NV12\n");
                                            result=0;
                                        } else {
                                            std::fprintf(stderr,"ERROR: Virtual camera does not advertise 1080p60\n");
                                            result=9;
                                        }
                                    } else {
                                        std::fprintf(stderr,"ERROR: Camera GetStreamCaps failed\n");
                                        result=10;
                                    }
                                    if(media) {
                                        if(media->pbFormat) CoTaskMemFree(media->pbFormat);
                                        if(media->pUnk) media->pUnk->Release();
                                        CoTaskMemFree(media);
                                    }
                                }else{
                                    std::fprintf(stderr,"IAMStreamConfig enumeration failed\n");
                                    result=8;
                                }
                                cfg->Release();
                            }
                            pin->Release();
                        }
                        pins->Release();
                    }
                    filter->Release();
                }
                VariantClear(&name);
                bag->Release();
                moniker->Release();
                break;
            }
            VariantClear(&name);
            bag->Release();
        }
        moniker->Release();
    }
    enumerator->Release();
    if(result==4)std::fprintf(stderr,"SolarisPKN MovilCam not enumerated\n");
    return result;
}
int main(){
    HRESULT hr=CoInitializeEx(nullptr,COINIT_APARTMENTTHREADED);
    if(FAILED(hr))return 1;
    const int result=test();
    CoUninitialize();
    return result;
}
