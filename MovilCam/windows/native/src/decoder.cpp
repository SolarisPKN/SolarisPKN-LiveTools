#include "decoder.hpp"
#include <mferror.h>
#include <wmcodecdsp.h>
#include <codecapi.h>
#include <d3d11.h>
#include <dxgi.h>
#include <algorithm>
#include <cstring>
#include <cstdio>
#include <limits>

#pragma comment(lib,"mfplat.lib")
#pragma comment(lib,"mfuuid.lib")
#pragma comment(lib,"mf.lib")
#pragma comment(lib,"d3d11.lib")
#pragma comment(lib,"wmcodecdspuuid.lib")

using Microsoft::WRL::ComPtr;
namespace movilcam {

void H264Decoder::reset() {
    if (transform_) {
        transform_->ProcessMessage(MFT_MESSAGE_COMMAND_FLUSH, 0);
        transform_->ProcessMessage(MFT_MESSAGE_NOTIFY_END_STREAMING, 0);
    }
    transform_.Reset(); dxgi_.Reset(); device_.Reset();
    nv12_.clear(); gpu_ = false; width_ = height_ = 0;
}

bool H264Decoder::configure_output() {
    if (!transform_) return false;
    // Check the decoder's supported output types instead of guessing.
    for (DWORD i = 0; i < 32; ++i) {
        ComPtr<IMFMediaType> out;
        HRESULT hr = transform_->GetOutputAvailableType(0, i, &out);
        if (FAILED(hr)) break;
        GUID subtype{};
        if (SUCCEEDED(out->GetGUID(MF_MT_SUBTYPE, &subtype)) &&
            subtype == MFVideoFormat_NV12) {
            if (SUCCEEDED(transform_->SetOutputType(0, out.Get(), 0))) return true;
        }
    }
    return false;
}

bool H264Decoder::open(DWORD width, DWORD height, DWORD fps, bool tryD3D11) {
    reset();
    if (!width || !height || (width & 1) || (height & 1)) return false;
    width_ = width; height_ = height; fps_ = fps;
    if (FAILED(CoCreateInstance(CLSID_CMSH264DecoderMFT, nullptr,
                                CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&transform_))))
        return false;
    ComPtr<IMFMediaType> input;
    if (FAILED(MFCreateMediaType(&input))) return false;
    if (FAILED(input->SetGUID(MF_MT_MAJOR_TYPE, MFMediaType_Video))) return false;
    if (FAILED(input->SetGUID(MF_MT_SUBTYPE, MFVideoFormat_H264_ES))) return false;
    if (FAILED(MFSetAttributeSize(input.Get(), MF_MT_FRAME_SIZE, width, height))) return false;
    if (FAILED(MFSetAttributeRatio(input.Get(), MF_MT_FRAME_RATE, fps, 1))) return false;
    if (FAILED(input->SetUINT32(MF_MT_INTERLACE_MODE,
                                MFVideoInterlace_MixedInterlaceOrProgressive))) return false;
    if (FAILED(transform_->SetInputType(0, input.Get(), 0))) return false;

    if (tryD3D11) {
        D3D_FEATURE_LEVEL level{};
        UINT flags = D3D11_CREATE_DEVICE_BGRA_SUPPORT | D3D11_CREATE_DEVICE_VIDEO_SUPPORT;
        HRESULT hr = D3D11CreateDevice(nullptr, D3D_DRIVER_TYPE_HARDWARE, nullptr,
                      flags, nullptr, 0, D3D11_SDK_VERSION, &device_, &level, nullptr);
        if (SUCCEEDED(hr)) {
            UINT token = 0;
            hr = MFCreateDXGIDeviceManager(&token, &dxgi_);
            if (SUCCEEDED(hr)) hr = dxgi_->ResetDevice(device_.Get(), token);
            if (SUCCEEDED(hr)) hr = transform_->ProcessMessage(
                MFT_MESSAGE_SET_D3D_MANAGER, reinterpret_cast<ULONG_PTR>(dxgi_.Get()));
            gpu_ = SUCCEEDED(hr);
        }
        if (!gpu_) { dxgi_.Reset(); device_.Reset(); }
    }
    if (!configure_output()) {
        // Windows decoder may not offer a concrete output type until SPS/PPS arrives.
        // Decode routine retries configuration after a stream-change notification.
        std::fprintf(stderr, "[MovilCam] Decoder output deferred until SPS/PPS.\n");
    }
    if (FAILED(transform_->ProcessMessage(MFT_MESSAGE_NOTIFY_BEGIN_STREAMING,0)))
        return false;
    if (FAILED(transform_->ProcessMessage(MFT_MESSAGE_NOTIFY_START_OF_STREAM,0)))
        return false;
    nv12_.resize(static_cast<std::size_t>(width_) * height_ * 3 / 2);
    return true;
}

bool H264Decoder::copy_nv12(IMFSample* sample, FrameRing& ring, LONGLONG pts) {
    if (!sample) return false;
    ComPtr<IMFMediaBuffer> buffer;
    if (FAILED(sample->ConvertToContiguousBuffer(&buffer))) return false;
    const std::size_t width = width_, height = height_;
    const std::size_t expected = width * height * 3 / 2;
    if (nv12_.size() != expected) nv12_.resize(expected);

    ComPtr<IMF2DBuffer2> two;
    if (SUCCEEDED(buffer.As(&two))) {
        BYTE *scan=nullptr, *base=nullptr;
        LONG pitch=0; DWORD capacity=0;
        HRESULT hr = two->Lock2DSize(MF2DBuffer_LockFlags_Read,
                                     &scan, &pitch, &base, &capacity);
        if (SUCCEEDED(hr)) {
            bool valid = pitch >= static_cast<LONG>(width) && pitch > 0 &&
                         scan >= base && std::size_t(scan-base) < capacity;
            const std::size_t padded = (height+15) & ~std::size_t(15);
            const std::size_t uvOffset = capacity >= std::size_t(pitch)*padded*3/2
                ? std::size_t(pitch)*padded : std::size_t(pitch)*height;
            if (valid && std::size_t(scan-base)+uvOffset+
                         std::size_t(pitch)*(height/2) <= capacity) {
                for (std::size_t y=0; y<height; ++y)
                    std::memcpy(nv12_.data()+y*width, scan+y*pitch, width);
                const BYTE* uv=scan+uvOffset;
                for (std::size_t y=0; y<height/2; ++y)
                    std::memcpy(nv12_.data()+width*height+y*width, uv+y*pitch, width);
            } else valid = false;
            two->Unlock2D();
            if (valid) return ring.publish(nv12_.data(),expected,pts);
        }
    }

    // Some Media Foundation output buffers are already contiguous CPU NV12.
    BYTE* raw=nullptr; DWORD maxLen=0, current=0;
    if (FAILED(buffer->Lock(&raw,&maxLen,&current))) return false;
    bool ok = false;
    if (current == expected) {
        std::memcpy(nv12_.data(),raw,expected);
        ok = true;
    } else if (current >= expected && current <= expected+width*48) {
        // Common 1080 -> 1088 padded output. Never assume contiguous planar UV
        // unless the extent agrees with the expected NV12 aligned-surface size.
        const std::size_t rows = (height+15) & ~std::size_t(15);
        if (current >= rows*width*3/2) {
            for (std::size_t y=0; y<height; ++y)
                std::memcpy(nv12_.data()+y*width,raw+y*width,width);
            for (std::size_t y=0; y<height/2; ++y)
                std::memcpy(nv12_.data()+width*height+y*width,
                            raw+rows*width+y*width,width);
            ok=true;
        }
    }
    buffer->Unlock();
    return ok && ring.publish(nv12_.data(),expected,pts);
}

bool H264Decoder::drain(FrameRing& ring, LONGLONG pts) {
    if (!transform_) return false;
    for (unsigned attempts=0; attempts<16; ++attempts) {
        MFT_OUTPUT_STREAM_INFO info{};
        if (FAILED(transform_->GetOutputStreamInfo(0, &info))) return false;
        ComPtr<IMFSample> supplied;
        ComPtr<IMFMediaBuffer> mem;
        if (!(info.dwFlags & (MFT_OUTPUT_STREAM_PROVIDES_SAMPLES|
                              MFT_OUTPUT_STREAM_CAN_PROVIDE_SAMPLES))) {
            if (FAILED(MFCreateSample(&supplied))) return false;
            const DWORD need = std::max<DWORD>(info.cbSize, width_*height_*3/2);
            if (FAILED(MFCreateMemoryBuffer(need, &mem))) return false;
            if (FAILED(supplied->AddBuffer(mem.Get()))) return false;
        }
        MFT_OUTPUT_DATA_BUFFER frame{};
        frame.dwStreamID = 0;
        frame.pSample = supplied.Get();
        DWORD status = 0;
        HRESULT hr = transform_->ProcessOutput(0,1,&frame,&status);
        ComPtr<IMFSample> produced;
        if (frame.pSample) {
            produced = frame.pSample;
            if (frame.pSample != supplied.Get()) frame.pSample->Release();
        }
        if (frame.pEvents) frame.pEvents->Release();
        if (hr == MF_E_TRANSFORM_STREAM_CHANGE) {
            if (!configure_output()) return false;
            continue;
        }
        if (hr == MF_E_TRANSFORM_NEED_MORE_INPUT) return true;
        if (FAILED(hr)) return false;
        if (produced) {
            LONGLONG time = pts;
            produced->GetSampleTime(&time);
            if (!copy_nv12(produced.Get(), ring, time)) return false;
        }
    }
    return true;
}

bool H264Decoder::decode(const std::uint8_t* data, std::size_t size,
                          LONGLONG pts, FrameRing& ring) {
    if (!transform_ || !data || size == 0 || size > std::numeric_limits<DWORD>::max())
        return false;
    ComPtr<IMFSample> sample;
    ComPtr<IMFMediaBuffer> buffer;
    if (FAILED(MFCreateSample(&sample))) return false;
    if (FAILED(MFCreateMemoryBuffer(static_cast<DWORD>(size), &buffer))) return false;
    BYTE* dst=nullptr;
    if (FAILED(buffer->Lock(&dst,nullptr,nullptr))) return false;
    std::memcpy(dst,data,size);
    buffer->Unlock();
    if (FAILED(buffer->SetCurrentLength(static_cast<DWORD>(size)))) return false;
    if (FAILED(sample->AddBuffer(buffer.Get()))) return false;
    sample->SetSampleTime(pts);
    sample->SetSampleDuration(10000000LL / fps_);
    HRESULT hr = transform_->ProcessInput(0,sample.Get(),0);
    if (hr == MF_E_NOTACCEPTING) {
        if (!drain(ring,pts)) return false;
        hr = transform_->ProcessInput(0,sample.Get(),0);
    }
    if (FAILED(hr)) return false;
    return drain(ring,pts);
}
} // namespace movilcam
