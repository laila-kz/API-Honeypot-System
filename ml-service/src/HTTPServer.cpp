#include "HTTPServer.hpp"
#include <httplib.h>
#include <nlohmann/json.hpp>
#include <iostream>

using json = nlohmann::json;

HTTPServer::HTTPServer(ONNXInference* inference)
    : m_inference(inference), m_running(false), m_port(5000) {}

HTTPServer::~HTTPServer() {
    stop();
}

void HTTPServer::handlePredict(const std::string& body, httplib::Response& res) {
    try {
        json request = json::parse(body);

        if (!request.contains("features")) {
            res.status = 400;
            res.set_content(json({
                {"error", "Missing 'features' field"}
            }).dump(), "application/json");
            return;
        }

        json features = request["features"];

        FeatureVector fv;
        fv.request_rate_per_ip = features.value("request_rate_per_ip", 0.0f);
        fv.endpoint_frequency = features.value("endpoint_frequency", 0.0f);
        fv.unique_endpoints_per_ip = features.value("unique_endpoints_per_ip", 0.0f);
        fv.payload_size = features.value("payload_size", 0.0f);
        fv.header_anomaly_score = features.value("header_anomaly_score", 0.0f);
        fv.time_between_requests = features.value("time_between_requests", 0.0f);
        fv.method_distribution = features.value("method_distribution", 0.0f);

        PredictionResult result = m_inference->predict(fv);

        res.status = 200;
        res.set_content(json({
            {"label", result.label},
            {"confidence", result.confidence}
        }).dump(), "application/json");

    } catch (const json::parse_error& e) {
        res.status = 400;
        res.set_content(json({
            {"error", "Invalid JSON format"},
            {"details", e.what()}
        }).dump(), "application/json");
    } catch (const std::exception& e) {
        res.status = 500;
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
        {"service", "C++ ONNX ML Service"},
        {"version", "1.1"}
    }).dump();
}

bool HTTPServer::start(int port) {
    m_port = port;
    m_running = true;
    m_server = std::make_unique<httplib::Server>();

    m_server->Post("/predict", [this](const httplib::Request& req, httplib::Response& res) {
        handlePredict(req.body, res);
    });

    m_server->Get("/health", [this](const httplib::Request&, httplib::Response& res) {
        std::string response;
        handleHealth(response);
        res.set_content(response, "application/json");
    });

    m_server->Get("/", [](const httplib::Request&, httplib::Response& res) {
        json info = {
            {"service", "Honeypot ML Service (C++ ONNX)"},
            {"version", "1.1"},
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

    m_serverThread = std::make_unique<std::thread>([this]() {
        std::cout << "[HTTP] C++ ML Service listening on port " << m_port << std::endl;
        m_server->listen("0.0.0.0", m_port);
    });

    return true;
}

void HTTPServer::stop() {
    if (m_server) {
        m_server->stop();
    }
    m_running = false;
    if (m_serverThread && m_serverThread->joinable()) {
        m_serverThread->join();
    }
}
