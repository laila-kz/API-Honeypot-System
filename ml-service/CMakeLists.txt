cmake_minimum_required(VERSION 3.20)
project(MLServiceCPP VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

option(ML_SERVICE_FETCH_HTTPLIB "Fetch cpp-httplib via FetchContent" ON)

add_executable(ml_service_cpp
    src/main.cpp
    src/ONNXInference.cpp
    src/HTTPServer.cpp
)

# ---------------------------------------------------------------------------
# Dependencies (prefer vcpkg CONFIG packages; fall back where needed)
# ---------------------------------------------------------------------------
find_package(onnxruntime CONFIG REQUIRED)
find_package(nlohmann_json CONFIG REQUIRED)
find_package(spdlog CONFIG REQUIRED)

# cpp-httplib: try package first, otherwise FetchContent header-only
find_package(httplib CONFIG QUIET)
if(NOT httplib_FOUND AND ML_SERVICE_FETCH_HTTPLIB)
    include(FetchContent)
    FetchContent_Declare(
        cpp_httplib
        GIT_REPOSITORY https://github.com/yhirose/cpp-httplib.git
        GIT_TAG        v0.14.3
    )
    FetchContent_MakeAvailable(cpp_httplib)
endif()

target_include_directories(ml_service_cpp PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

target_link_libraries(ml_service_cpp PRIVATE
    onnxruntime::onnxruntime
    nlohmann_json::nlohmann_json
    spdlog::spdlog
)

if(TARGET httplib::httplib)
    target_link_libraries(ml_service_cpp PRIVATE httplib::httplib)
elseif(TARGET cpp-httplib)
    target_link_libraries(ml_service_cpp PRIVATE cpp-httplib)
elseif(TARGET httplib)
    target_link_libraries(ml_service_cpp PRIVATE httplib)
endif()

# Platform-specific system libraries
if(WIN32)
    target_link_libraries(ml_service_cpp PRIVATE ws2_32 crypt32 user32 advapi32)
elseif(UNIX AND NOT APPLE)
    find_package(Threads REQUIRED)
    target_link_libraries(ml_service_cpp PRIVATE Threads::Threads)
elseif(APPLE)
    find_package(Threads REQUIRED)
    target_link_libraries(ml_service_cpp PRIVATE Threads::Threads)
endif()

# Default ONNX model path (overridable via ONNX_MODEL_PATH at runtime)
set(MODEL_PATH "${CMAKE_CURRENT_SOURCE_DIR}/models/random_forest.onnx")
file(TO_CMAKE_PATH "${MODEL_PATH}" MODEL_PATH_NORMALIZED)

target_compile_definitions(ml_service_cpp PRIVATE
    MODEL_PATH="${MODEL_PATH_NORMALIZED}"
)

if(MSVC)
    target_compile_definitions(ml_service_cpp PRIVATE _CRT_SECURE_NO_WARNINGS)
    target_compile_options(ml_service_cpp PRIVATE /W3 /EHsc)
else()
    target_compile_options(ml_service_cpp PRIVATE -Wall -Wextra -Wno-unused-parameter)
endif()

set_target_properties(ml_service_cpp PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
    RUNTIME_OUTPUT_DIRECTORY_RELEASE "${CMAKE_BINARY_DIR}/bin/Release"
    RUNTIME_OUTPUT_DIRECTORY_DEBUG "${CMAKE_BINARY_DIR}/bin/Debug"
)

message(STATUS "ML Service model path: ${MODEL_PATH_NORMALIZED}")
