#pragma once
#include "ring.hpp"
#include <cstdint>
#include <cstddef>
#include <vector>
#include <wrl/client.h>
#include <mftransform.h>
#include <d3d11.h>
#include <mfidl.h>
#include <mfapi.h>
namespace movilcam {
enum class GpuChoice { Automatic, Intel, Nvidia, Amd, Software };
class H264Decoder {
    Microsoft::WRL::ComPtr<IMFTransform> transform_;
    Microsoft::WRL::ComPtr<ID3D11Device> device_;
    Microsoft::WRL::ComPtr<IMFDXGIDeviceManager> dxgi_;
    DWORD width_ = 0, height_ = 0, fps_ = 30;
    bool gpu_ = false;
    std::vector<std::uint8_t> nv12_;
    bool configure_output();
    bool drain(FrameRing&, LONGLONG pts100ns);
    bool copy_nv12(IMFSample* sample, FrameRing&, LONGLONG pts100ns);
public:
    H264Decoder() = default;
    H264Decoder(const H264Decoder&) = delete;
    H264Decoder& operator=(const H264Decoder&) = delete;
    bool open(DWORD width, DWORD height, DWORD fps, GpuChoice gpu);
    bool decode(const std::uint8_t* data, std::size_t bytes, LONGLONG pts100ns,
                FrameRing& ring);
    void reset();
    bool gpu_enabled() const noexcept { return gpu_; }
};
}
