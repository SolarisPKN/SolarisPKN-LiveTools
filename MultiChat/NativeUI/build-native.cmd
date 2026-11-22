@echo off
setlocal
set "MSVC=C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\VC\Auxiliary\Build\vcvars64.bat"
if not exist "%MSVC%" exit /b 2
call "%MSVC%" >NUL || exit /b 3
cd /d "%~dp0"
cl /nologo /O2 /EHsc /utf-8 /std:c++17 /D_UNICODE /DUNICODE /DNOMINMAX native_multichat.cpp /link /SUBSYSTEM:WINDOWS /OUT:"..\MultiChat-Native.exe" winhttp.lib gdiplus.lib user32.lib gdi32.lib comctl32.lib || exit /b 4
exit /b 0
