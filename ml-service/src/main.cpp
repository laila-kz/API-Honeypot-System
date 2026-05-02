#include "ONNXInference.hpp"
#include "HTTPServer.hpp"
#include <iostream>
#include <csignal>
#include <atomic>
#include <chrono>
#include <thread>

std::atomic<bool> running(true);

void signalHandler(int signal) {
    std::cout << "\n[Main] Received signal " << signal << ", shutting down..." << std::endl;
    running = false;
}

int main(int argc, char* argv[]) {
    std::cout << "=== C++ ONNX ML Service ===" << std::endl;
    std::cout << "Starting Honeypot ML Inference Service..." << std::endl;
    
    // Set up signal handlers
    signal(SIGINT, signalHandler);
    signal(SIGTERM, signalHandler);
    
    // Model path (can be overridden by environment variable)
    std::string modelPath = MODEL_PATH;
    const char* envPath = std::getenv("ONNX_MODEL_PATH");
    if (envPath != nullptr) {
        modelPath = std::string(envPath);
    }
    
    std::cout << "[Main] Loading ONNX model from: " << modelPath << std::endl;
    
    // Initialize inference engine
    auto inference = std::make_unique<ONNXInference>();
    if (!inference->initialize(modelPath)) {
        std::cerr << "[Main] Failed to initialize ONNX model!" << std::endl;
        std::cerr << "[Main] Make sure the model exists at: " << modelPath << std::endl;
        return 1;
    }
    
    std::cout << "[Main] Model loaded successfully!" << std::endl;
    
    // Start HTTP server
    HTTPServer server(inference.get());
    int port = 5000;
    const char* envPort = std::getenv("ML_SERVICE_PORT");
    if (envPort != nullptr) {
        port = std::atoi(envPort);
    }
    
    if (!server.start(port)) {
        std::cerr << "[Main] Failed to start HTTP server!" << std::endl;
        return 1;
    }
    
    std::cout << "[Main] Service running on http://localhost:" << port << std::endl;
    std::cout << "[Main] Press Ctrl+C to stop" << std::endl;
    
    // Keep running
    while (running) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
    
    std::cout << "[Main] Shutting down..." << std::endl;
    server.stop();
    
    return 0;
}