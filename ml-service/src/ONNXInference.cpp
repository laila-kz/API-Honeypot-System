#include "ONNXInference.hpp"
#include <iostream>
#include <algorithm>
#include <chrono>
#include <cstring>
#include <cmath>

ONNXInference::ONNXInference() : m_initialized(false) {}

ONNXInference::~ONNXInference() {
    // smart pointers handle cleanup
}

bool ONNXInference::initialize(const std::string& modelPath) {
    try {
        m_env = std::make_unique<Ort::Env>(ORT_LOGGING_LEVEL_WARNING, "MLService");

        Ort::SessionOptions sessionOptions;
        sessionOptions.SetIntraOpNumThreads(1);
        sessionOptions.SetGraphOptimizationLevel(GraphOptimizationLevel::ORT_ENABLE_EXTENDED);

        // ✅ FIX: convert to wide string for Windows
        std::wstring wModelPath(modelPath.begin(), modelPath.end());

        m_session = std::make_unique<Ort::Session>(
            *m_env,
            wModelPath.c_str(),
            sessionOptions
        );

        Ort::AllocatorWithDefaultOptions allocator;

        size_t numInputNodes = m_session->GetInputCount();

        m_inputNameStrs.clear();
        m_inputNames.clear();

        for (size_t i = 0; i < numInputNodes; i++) {
            auto name = m_session->GetInputNameAllocated(i, allocator);
            m_inputNameStrs.emplace_back(name.get());
        }

        for (auto& s : m_inputNameStrs) {
            m_inputNames.push_back(s.c_str());
        }

        // same fix applies to outputs
        size_t numOutputNodes = m_session->GetOutputCount();

        m_outputNameStrs.clear();
        m_outputNames.clear();

        for (size_t i = 0; i < numOutputNodes; i++) {
            auto name = m_session->GetOutputNameAllocated(i, allocator);
            m_outputNameStrs.emplace_back(name.get());
        }

        for (auto& s : m_outputNameStrs) {
            m_outputNames.push_back(s.c_str());
        }

        m_initialized = true;
        return true;

    } catch (const Ort::Exception& e) {
        std::cerr << "[ONNX Error] " << e.what() << std::endl;
        return false;
    }
}

PredictionResult ONNXInference::predict(const FeatureVector& features) {
    if (!m_initialized) {
        return {"normal", 0.5f};
    }

    try {
        std::vector<float> inputData = features.toVector();

        std::vector<int64_t> inputShape = {1, (int64_t)inputData.size()};

        Ort::Value inputTensor = Ort::Value::CreateTensor<float>(
            *m_memoryInfo,
            inputData.data(),
            inputData.size(),
            inputShape.data(),
            inputShape.size()
        );

        auto start = std::chrono::high_resolution_clock::now();

        auto outputTensors = m_session->Run(
            Ort::RunOptions{nullptr},
            m_inputNames.data(),
            &inputTensor,
            1,
            m_outputNames.data(),
            m_outputNames.size()
        );

        auto end = std::chrono::high_resolution_clock::now();

        std::cout << "[ONNX] Inference time: "
                  << std::chrono::duration_cast<std::chrono::microseconds>(end - start).count()
                  << " μs" << std::endl;

        return processOutput(outputTensors[0]);

    } catch (const Ort::Exception& e) {
        std::cerr << "[ONNX Prediction Error] " << e.what() << std::endl;
        return {"normal", 0.5f};
    }
}

PredictionResult ONNXInference::processOutput(const Ort::Value& outputTensor) {
    auto info = outputTensor.GetTensorTypeAndShapeInfo();
    size_t count = info.GetElementCount();

    const float* data = outputTensor.GetTensorData<float>();

    std::vector<float> output(data, data + count);

    PredictionResult result;

    if (output.size() == 1) {
        float score = output[0];
        result.confidence = std::abs(score);

        if (score > 0.6f) result.label = "attack";
        else if (score > 0.3f) result.label = "suspicious";
        else result.label = "normal";
    } else {
        float sum = 0.0f;
        for (float v : output) sum += std::exp(v);

        std::vector<float> probs(output.size());
        for (size_t i = 0; i < output.size(); i++) {
            probs[i] = std::exp(output[i]) / sum;
        }

        auto maxIt = std::max_element(probs.begin(), probs.end());
        int idx = std::distance(probs.begin(), maxIt);

        result.label = (idx < m_labels.size()) ? m_labels[idx] : "normal";
        result.confidence = *maxIt;
    }

    result.confidence = std::clamp(result.confidence, 0.5f, 0.99f);

    return result;
}