#ifndef ONNX_INFERENCE_HPP
#define ONNX_INFERENCE_HPP

#include <onnxruntime_cxx_api.h>
#include <vector>
#include <string>
#include <memory>
#include <array>

struct FeatureVector {
    float request_rate_per_ip;
    float endpoint_frequency;
    float unique_endpoints_per_ip;
    float payload_size;
    float header_anomaly_score;
    float time_between_requests;
    float method_distribution;

    std::vector<float> toVector() const {
        return {
            request_rate_per_ip,
            endpoint_frequency,
            unique_endpoints_per_ip,
            payload_size,
            header_anomaly_score,
            time_between_requests,
            method_distribution
        };
    }
};

struct PredictionResult {
    std::string label;  // "normal", "suspicious", "attack"
    float confidence;
};

class ONNXInference {
public:
    ONNXInference();
    ~ONNXInference();

    bool initialize(const std::string& modelPath);
    PredictionResult predict(const FeatureVector& features);
    bool isInitialized() const { return m_initialized; }

private:
    std::unique_ptr<Ort::Env> m_env;
    std::unique_ptr<Ort::Session> m_session;
    std::unique_ptr<Ort::MemoryInfo> m_memoryInfo;
    bool m_initialized;

    std::vector<std::string> m_inputNameStrs;
    std::vector<std::string> m_outputNameStrs;
    std::vector<const char*> m_inputNames;
    std::vector<const char*> m_outputNames;
    std::vector<int64_t> m_expectedInputDims;

    std::vector<std::string> m_labels = {"normal", "suspicious", "attack"};

    PredictionResult processOutputs(std::vector<Ort::Value>& outputTensors);
    PredictionResult processFloatTensor(const Ort::Value& outputTensor);
    PredictionResult processInt64Labels(const Ort::Value& outputTensor);
    static PredictionResult fallbackResult();
};

#endif // ONNX_INFERENCE_HPP
