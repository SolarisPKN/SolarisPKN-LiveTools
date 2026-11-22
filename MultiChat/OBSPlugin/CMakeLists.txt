cmake_minimum_required(VERSION 3.28)
project(obs-multichat VERSION 0.1.0 LANGUAGES CXX)

# Requires the OBS Studio development SDK / libobs CMake package.
# On Windows set -Dlibobs_DIR=<path-to-libobs-cmake-config>.
if(NOT WIN32)
  message(FATAL_ERROR "The initial MultiChat OBS plugin supports Windows only (WinHTTP + OBS GDI+ text).")
endif()

find_package(libobs CONFIG REQUIRED)
add_library(obs-multichat MODULE obs-multichat.cpp)
target_compile_features(obs-multichat PRIVATE cxx_std_17)
target_compile_definitions(obs-multichat PRIVATE UNICODE _UNICODE WIN32_LEAN_AND_MEAN NOMINMAX)
target_link_libraries(obs-multichat PRIVATE OBS::libobs winhttp)
set_target_properties(obs-multichat PROPERTIES PREFIX "" OUTPUT_NAME "obs-multichat")
if(MSVC)
  target_compile_options(obs-multichat PRIVATE /W4 /EHsc)
endif()
install(TARGETS obs-multichat RUNTIME DESTINATION "obs-plugins/64bit" LIBRARY DESTINATION "obs-plugins/64bit")
