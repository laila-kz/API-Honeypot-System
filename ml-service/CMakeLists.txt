cmake_minimum_required(VERSION 3.20)
project(MLServiceCPP VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Add executable
add_executable(ml_service_cpp
    src/main.cpp
    src/ONNXInference.cpp
    src/HTTPServer.cpp
)

# Find packages via vcpkg
find_package(onnxruntime CONFIG REQUIRED)
find_package(nlohmann_json CONFIG REQUIRED)
find_package(spdlog CONFIG REQUIRED)

# Include project headers
target_include_directories(ml_service_cpp PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# Link libraries
target_link_libraries(ml_service_cpp PRIVATE
    onnxruntime::onnxruntime
    nlohmann_json::nlohmann_json
    spdlog::spdlog
    ws2_32
    crypt32
    user32
    advapi32
)

# Define model path
set(MODEL_PATH "${CMAKE_CURRENT_SOURCE_DIR}/models/random_forest.onnx")

target_compile_definitions(ml_service_cpp PRIVATE
    MODEL_PATH="${MODEL_PATH}"
    _CRT_SECURE_NO_WARNINGS
)

# Output directory
set_target_properties(ml_service_cpp PROPERTIES
    RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
)