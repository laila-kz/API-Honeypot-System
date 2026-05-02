@echo off
REM Windows build script for C++ ML Service

echo ========================================
echo Building C++ ONNX ML Service for Windows
echo ========================================

REM Set vcpkg path
set VCPKG_ROOT=C:\vcpkg
set PATH=%VCPKG_ROOT%;%PATH%

REM Create build directory
if not exist build mkdir build
cd build

REM Configure with CMake
echo Configuring CMake...
cmake .. ^
    -DCMAKE_TOOLCHAIN_FILE=%VCPKG_ROOT%\scripts\buildsystems\vcpkg.cmake ^
    -DVCPKG_TARGET_TRIPLET=x64-windows ^
    -DCMAKE_BUILD_TYPE=Release

REM Build the project
echo Building project...
cmake --build . --config Release

echo.
echo ========================================
echo Build complete!
echo Executable: build\bin\Release\ml_service_cpp.exe
echo ========================================

cd ..