cmake_minimum_required(VERSION 3.23)
project(SolarisPKNMovilCamBridge LANGUAGES CXX)
if(NOT WIN32)
  message(FATAL_ERROR "MovilCam native bridge targets Windows 10/11 x64.")
endif()
add_library(movilcam_protocol INTERFACE)
target_include_directories(movilcam_protocol INTERFACE "${CMAKE_CURRENT_SOURCE_DIR}/include")
add_executable(movilcam-protocol-test tests/protocol_test.cpp)
target_compile_features(movilcam-protocol-test PRIVATE cxx_std_17)
target_link_libraries(movilcam-protocol-test PRIVATE movilcam_protocol)
enable_testing()
add_test(NAME protocol COMMAND movilcam-protocol-test)

add_executable(MovilCamBridge src/main.cpp src/ring.cpp src/decoder.cpp)
target_compile_features(MovilCamBridge PRIVATE cxx_std_17)
target_compile_definitions(MovilCamBridge PRIVATE WIN32_LEAN_AND_MEAN NOMINMAX _WIN32_WINNT=0x0A00)
target_link_libraries(MovilCamBridge PRIVATE movilcam_protocol
  ws2_32 ole32 mfplat mf mfuuid d3d11 wmcodecdspuuid)
if(MSVC)
  target_compile_options(MovilCamBridge PRIVATE /W4 /EHsc /permissive-)
  target_compile_options(movilcam-protocol-test PRIVATE /W4 /EHsc /permissive-)
endif()

# Compatibility camera for Windows 10, requires Microsoft's open sample DirectShow
# base classes at build time. They are NOT distributed in our repository.
set(MOVILCAM_DSHOW_BASE "" CACHE PATH "Path to Windows-classic-samples DirectShow baseclasses")
if(MOVILCAM_DSHOW_BASE)
  if(NOT EXISTS "${MOVILCAM_DSHOW_BASE}/streams.h")
    message(FATAL_ERROR "DirectShow baseclasses folder missing streams.h")
  endif()
  file(GLOB BASECLASSES_CPP CONFIGURE_DEPENDS "${MOVILCAM_DSHOW_BASE}/*.cpp")
  add_library(movilcam_dshow_base STATIC ${BASECLASSES_CPP})
  target_include_directories(movilcam_dshow_base PUBLIC "${MOVILCAM_DSHOW_BASE}")
  target_compile_definitions(movilcam_dshow_base PRIVATE
      WIN32_LEAN_AND_MEAN NOMINMAX _WIN32_WINNT=0x0A00)
  if(MSVC)
    target_compile_options(movilcam_dshow_base PRIVATE /w /EHsc)
  endif()
  target_link_libraries(movilcam_dshow_base PUBLIC
      strmiids winmm ole32 oleaut32 uuid)
  add_library(MovilCamVirtualCamera SHARED
      virtualcam/virtualcam.cpp virtualcam/exports.def)
  target_compile_features(MovilCamVirtualCamera PRIVATE cxx_std_17)
  target_compile_definitions(MovilCamVirtualCamera PRIVATE
      WIN32_LEAN_AND_MEAN NOMINMAX _WIN32_WINNT=0x0A00)
  target_link_libraries(MovilCamVirtualCamera PRIVATE movilcam_dshow_base
      movilcam_protocol strmiids ole32 oleaut32 uuid)
  if(MSVC)
    target_compile_options(MovilCamVirtualCamera PRIVATE /W4 /EHsc)
  endif()
endif()
