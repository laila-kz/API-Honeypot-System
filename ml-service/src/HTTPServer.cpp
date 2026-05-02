#include "HTTPServer.hpp"
#include <httplib.h>
#include <nlohmann/json.hpp>
#include <windows.h>  // For Windows-specific threading
// Add at the top of HTTPServer.cpp for Windows
#ifdef _WIN32
#define _WINSOCK_DEPRECATED_NO_WARNINGS
#include <winsock2.h>
#pragma comment(lib, "ws2_32.lib")
#endif

using json = nlohmann::json;

// Windows thread wrapper
class WindowsThread {
private:
    HANDLE m_thread;
    DWORD m_threadId;
    
public:
    template<typename Func>
    WindowsThread(Func func) {
        m_thread = CreateThread(
            nullptr,
            0,
            [](LPVOID param) -> DWORD {
                auto* f = static_cast<std::function<void()>*>(param);
                (*f)();
                delete f;
                return 0;
            },
            new std::function<void()>(func),
            0,
            &m_threadId
        );
    }
    
    ~WindowsThread() {
        if (m_thread) CloseHandle(m_thread);
    }
    
    void join() {
        if (m_thread) {
            WaitForSingleObject(m_thread, INFINITE);
        }
    }
};

HTTPServer::HTTPServer(ONNXInference* inference) 
    : m_inference(inference), m_running(false), m_port(5000) {}

HTTPServer::~HTTPServer() {
    stop();
}

void HTTPServer::handlePredict(const std::string& body, httplib::Response& res) {
    try {
        // Parse JSON request
        json request = json::parse(body);
        
        if (!request.contains("features")) {
            res.set_content(json({
                {"error", "Missing 'features' field"}
            }).dump(), "application/json");
            return;
        }
        
        json features = request["features"];
        
        // Extract feature vector
        FeatureVector fv;
        fv.request_rate_per_ip = features.value("request_rate_per_ip", 0.0f);
        fv.endpoint_frequency = features.value("endpoint_frequency", 0.0f);
        fv.unique_endpoints_per_ip = features.value("unique_endpoints_per_ip", 0.0f);
        fv.payload_size = features.value("payload_size", 0.0f);
        fv.header_anomaly_score = features.value("header_anomaly_score", 0.0f);
        fv.time_between_requests = features.value("time_between_requests", 0.0f);
        fv.method_distribution = features.value("method_distribution", 0.0f);
        
        // Get prediction
        PredictionResult result = m_inference->predict(fv);
        
        // Build response
        res.set_content(json({
            {"label", result.label},
            {"confidence", result.confidence}
        }).dump(), "application/json");
        
    } catch (const json::parse_error& e) {
        res.set_content(json({
            {"error", "Invalid JSON format"},
            {"details", e.what()}
        }).dump(), "application/json");
    } catch (const std::exception& e) {
        res.set_content(json({
            {"error", "Prediction failed"},
            {"details", e.what()}
        }).dump(), "application/json");
    }
}

void HTTPServer::handleHealth(std::string& response) {
    response = json({
        {"status", "healthy"},
        {"model_loaded", m_inference->isInitialized()},
        {"service", "C++ ONNX ML Service (Windows)"},
        {"version", "1.0"}
    }).dump();
}

bool HTTPServer::start(int port) {
    m_port = port;
    m_running = true;
    
    // Create server thread
    m_serverThread = std::make_unique<std::thread>([this]() {
        httplib::Server svr;
        
        // Prediction endpoint
        svr.Post("/predict", [this](const httplib::Request& req, httplib::Response& res) {
            handlePredict(req.body, res);
        });
        
        // Health check endpoint
        svr.Get("/health", [this](const httplib::Request& req, httplib::Response& res) {
            std::string response;
            handleHealth(response);
            res.set_content(response, "application/json");
        });
        
        // Root endpoint
        svr.Get("/", [](const httplib::Request& req, httplib::Response& res) {
            json info = {
                {"service", "Honeypot ML Service (C++ Windows)"},
                {"version", "1.0"},
                {"model_type", "Random Forest (ONNX)"},
                {"features", {
                    "request_rate_per_ip",
                    "endpoint_frequency", 
                    "unique_endpoints_per_ip",
                    "payload_size",
                    "header_anomaly_score",
                    "time_between_requests",
                    "method_distribution"
                }},
                {"endpoints", {
                    {"POST /predict", "Get prediction for feature vector"},
                    {"GET /health", "Health check"},
                    {"GET /", "Service information"}
                }}
            };
            res.set_content(info.dump(2), "application/json");
        });
        
        std::cout << "[HTTP] C++ ML Service listening on port " << m_port << std::endl;
        svr.listen("0.0.0.0", m_port);
    });
    
    return true;
}

void HTTPServer::stop() {
    if (m_running && m_serverThread && m_serverThread->joinable()) {
        m_running = false;
        // Note: httplib doesn't have graceful shutdown on Windows easily
        // We'll just detach to avoid crash on exit
        m_serverThread->detach();
    }
}