#include "ONNXInference.hpp"
#include "HTTPServer.hpp"
#include <iostream>
#include <csignal>
#include <atomic>
#include <chrono>
#include <thread>
#include <cstdlib>

std::atomic<bool> running(true);

void signalHandler(int signal) {
    std::cout << "\n[Main] Received signal " << signal << ", shutting down..." << std::endl;
    running = false;
}

int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    std::cout << "=== C++ ONNX ML Service ===" << std::endl;
    std::cout << "Starting Honeypot ML Inference Service..." << std::endl;

    signal(SIGINT, signalHandler);
    signal(SIGTERM, signalHandler);

    std::string modelPath = MODEL_PATH;
    const char* envPath = std::getenv("ONNX_MODEL_PATH");
    if (envPath != nullptr && envPath[0] != '\0') {
        modelPath = std::string(envPath);
    }

    std::cout << "[Main] Loading ONNX model from: " << modelPath << std::endl;

    auto inference = std::make_unique<ONNXInference>();
    if (!inference->initialize(modelPath)) {
        std::cerr << "[Main] Failed to initialize ONNX model!" << std::endl;
        std::cerr << "[Main] Ensure the model exists at: " << modelPath << std::endl;
        std::cerr << "[Main] Or set ONNX_MODEL_PATH to a valid .onnx file." << std::endl;
        return 1;
    }

    std::cout << "[Main] Model loaded successfully!" << std::endl;

    HTTPServer server(inference.get());
    int port = 5000;
    const char* envPort = std::getenv("ML_SERVICE_PORT");
    if (envPort != nullptr && envPort[0] != '\0') {
        port = std::atoi(envPort);
        if (port <= 0 || port > 65535) {
            std::cerr << "[Main] Invalid ML_SERVICE_PORT; defaulting to 5000" << std::endl;
            port = 5000;
        }
    }

    if (!server.start(port)) {
        std::cerr << "[Main] Failed to start HTTP server!" << std::endl;
        return 1;
    }

    std::cout << "[Main] Service running on http://0.0.0.0:" << port << std::endl;
    std::cout << "[Main] Press Ctrl+C to stop" << std::endl;

    while (running) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }

    std::cout << "[Main] Shutting down..." << std::endl;
    server.stop();

    return 0;
}
