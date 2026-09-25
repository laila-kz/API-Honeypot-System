#include "ONNXInference.hpp"
#include <iostream>
#include <algorithm>
#include <chrono>
#include <cstring>
#include <cmath>
#include <numeric>
#include <sstream>

#ifdef _WIN32
#include <windows.h>
#endif

namespace {

#ifdef _WIN32
std::wstring toWidePath(const std::string& path) {
    if (path.empty()) {
        return std::wstring();
    }
    int sizeNeeded = MultiByteToWideChar(CP_UTF8, 0, path.c_str(), -1, nullptr, 0);
    if (sizeNeeded <= 0) {
        return std::wstring(path.begin(), path.end());
    }
    std::wstring wide(static_cast<size_t>(sizeNeeded), L'\0');
    MultiByteToWideChar(CP_UTF8, 0, path.c_str(), -1, &wide[0], sizeNeeded);
    if (!wide.empty() && wide.back() == L'\0') {
        wide.pop_back();
    }
    return wide;
}
#endif

std::string shapeToString(const std::vector<int64_t>& shape) {
    std::ostringstream oss;
    oss << "[";
    for (size_t i = 0; i < shape.size(); ++i) {
        if (i) oss << ",";
        oss << shape[i];
    }
    oss << "]";
    return oss.str();
}

} // namespace

ONNXInference::ONNXInference() : m_initialized(false) {}

ONNXInference::~ONNXInference() = default;

PredictionResult ONNXInference::fallbackResult() {
    return {"normal", 0.5f};
}

bool ONNXInference::initialize(const std::string& modelPath) {
    try {
        m_env = std::make_unique<Ort::Env>(ORT_LOGGING_LEVEL_WARNING, "MLService");
        m_memoryInfo = std::make_unique<Ort::MemoryInfo>(
            Ort::MemoryInfo::CreateCpu(OrtArenaAllocator, OrtMemTypeDefault));

        Ort::SessionOptions sessionOptions;
        sessionOptions.SetIntraOpNumThreads(1);
        sessionOptions.SetGraphOptimizationLevel(GraphOptimizationLevel::ORT_ENABLE_EXTENDED);

#ifdef _WIN32
        std::wstring wModelPath = toWidePath(modelPath);
        m_session = std::make_unique<Ort::Session>(
            *m_env, wModelPath.c_str(), sessionOptions);
#else
        m_session = std::make_unique<Ort::Session>(
            *m_env, modelPath.c_str(), sessionOptions);
#endif

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

        // Cache expected input dims for shape validation (dynamic dims as -1).
        if (numInputNodes > 0) {
            auto typeInfo = m_session->GetInputTypeInfo(0);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            m_expectedInputDims = tensorInfo.GetShape();
            std::cout << "[ONNX] Input shape: " << shapeToString(m_expectedInputDims) << std::endl;
        }

        size_t numOutputNodes = m_session->GetOutputCount();
        m_outputNameStrs.clear();
        m_outputNames.clear();
        for (size_t i = 0; i < numOutputNodes; i++) {
            auto name = m_session->GetOutputNameAllocated(i, allocator);
            m_outputNameStrs.emplace_back(name.get());
            auto typeInfo = m_session->GetOutputTypeInfo(i);
            if (typeInfo.GetONNXType() == ONNX_TYPE_TENSOR) {
                auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
                std::cout << "[ONNX] Output[" << i << "] name=" << m_outputNameStrs.back()
                          << " shape=" << shapeToString(tensorInfo.GetShape())
                          << " type=" << static_cast<int>(tensorInfo.GetElementType())
                          << std::endl;
            } else {
                std::cout << "[ONNX] Output[" << i << "] name=" << m_outputNameStrs.back()
                          << " (non-tensor type " << static_cast<int>(typeInfo.GetONNXType())
                          << ")" << std::endl;
            }
        }
        for (auto& s : m_outputNameStrs) {
            m_outputNames.push_back(s.c_str());
        }

        m_initialized = true;
        return true;

    } catch (const Ort::Exception& e) {
        std::cerr << "[ONNX Error] " << e.what() << std::endl;
        m_initialized = false;
        return false;
    } catch (const std::exception& e) {
        std::cerr << "[ONNX Error] " << e.what() << std::endl;
        m_initialized = false;
        return false;
    }
}

PredictionResult ONNXInference::predict(const FeatureVector& features) {
    if (!m_initialized || !m_session || !m_memoryInfo) {
        return fallbackResult();
    }

    try {
        std::vector<float> inputData = features.toVector();

        // Build input shape: prefer model metadata when batch/feature dims are fixed.
        std::vector<int64_t> inputShape = {1, static_cast<int64_t>(inputData.size())};
        if (m_expectedInputDims.size() == 2) {
            int64_t featDim = m_expectedInputDims[1];
            if (featDim > 0 && static_cast<size_t>(featDim) != inputData.size()) {
                std::cerr << "[ONNX] Feature count mismatch: got " << inputData.size()
                          << " expected " << featDim
                          << " — padding/truncating to model shape" << std::endl;
                inputData.resize(static_cast<size_t>(featDim), 0.0f);
                inputShape[1] = featDim;
            } else if (featDim > 0) {
                inputShape[1] = featDim;
            }
            if (m_expectedInputDims[0] > 0) {
                inputShape[0] = m_expectedInputDims[0];
            }
        } else if (m_expectedInputDims.size() == 1 && m_expectedInputDims[0] > 0) {
            inputShape = {m_expectedInputDims[0]};
            inputData.resize(static_cast<size_t>(m_expectedInputDims[0]), 0.0f);
        }

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
                  << " us" << std::endl;

        return processOutputs(outputTensors);

    } catch (const Ort::Exception& e) {
        std::cerr << "[ONNX Prediction Error] " << e.what() << std::endl;
        return fallbackResult();
    } catch (const std::exception& e) {
        std::cerr << "[ONNX Prediction Error] " << e.what() << std::endl;
        return fallbackResult();
    }
}

PredictionResult ONNXInference::processOutputs(std::vector<Ort::Value>& outputTensors) {
    if (outputTensors.empty()) {
        std::cerr << "[ONNX] No output tensors returned" << std::endl;
        return fallbackResult();
    }

    // Prefer a float probability / logit tensor when multiple outputs exist
    // (common for sklearn→ONNX: label int64 + probability float).
    int floatIdx = -1;
    int intIdx = -1;
    for (size_t i = 0; i < outputTensors.size(); ++i) {
        if (!outputTensors[i].IsTensor()) {
            continue;
        }
        auto info = outputTensors[i].GetTensorTypeAndShapeInfo();
        auto elem = info.GetElementType();
        if (elem == ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT ||
            elem == ONNX_TENSOR_ELEMENT_DATA_TYPE_DOUBLE) {
            floatIdx = static_cast<int>(i);
        } else if (elem == ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64 ||
                   elem == ONNX_TENSOR_ELEMENT_DATA_TYPE_INT32) {
            intIdx = static_cast<int>(i);
        }
    }

    if (floatIdx >= 0) {
        return processFloatTensor(outputTensors[static_cast<size_t>(floatIdx)]);
    }
    if (intIdx >= 0) {
        return processInt64Labels(outputTensors[static_cast<size_t>(intIdx)]);
    }

    // Last resort: try first tensor as float.
    try {
        return processFloatTensor(outputTensors[0]);
    } catch (...) {
        std::cerr << "[ONNX] Unable to interpret model outputs — using fallback" << std::endl;
        return fallbackResult();
    }
}

PredictionResult ONNXInference::processFloatTensor(const Ort::Value& outputTensor) {
    auto info = outputTensor.GetTensorTypeAndShapeInfo();
    auto shape = info.GetShape();
    size_t count = info.GetElementCount();

    std::cout << "[ONNX] Float output shape=" << shapeToString(shape)
              << " elements=" << count << std::endl;

    if (count == 0) {
        std::cerr << "[ONNX] Empty float output tensor" << std::endl;
        return fallbackResult();
    }

    const float* data = outputTensor.GetTensorData<float>();
    std::vector<float> output(data, data + count);

    PredictionResult result = fallbackResult();

    if (output.size() == 1) {
        float score = output[0];
        // Treat as probability or decision score in [0,1] / unbounded.
        float conf = score;
        if (conf < 0.0f) conf = 0.0f;
        if (conf > 1.0f) conf = 1.0f / (1.0f + std::exp(-score)); // sigmoid if logit-like

        result.confidence = conf;
        if (conf > 0.6f) result.label = "attack";
        else if (conf > 0.3f) result.label = "suspicious";
        else result.label = "normal";
    } else {
        // Softmax over class logits / raw scores.
        float maxLogit = *std::max_element(output.begin(), output.end());
        float sum = 0.0f;
        std::vector<float> probs(output.size());
        for (size_t i = 0; i < output.size(); i++) {
            probs[i] = std::exp(output[i] - maxLogit);
            sum += probs[i];
        }
        if (sum <= 0.0f) {
            return fallbackResult();
        }
        for (float& p : probs) {
            p /= sum;
        }

        // If values already look like probabilities (sum≈1, all in [0,1]), use as-is.
        float rawSum = std::accumulate(output.begin(), output.end(), 0.0f);
        bool alreadyProbs = std::fabs(rawSum - 1.0f) < 0.05f;
        if (alreadyProbs) {
            bool inRange = true;
            for (float v : output) {
                if (v < -0.01f || v > 1.01f) {
                    inRange = false;
                    break;
                }
            }
            if (inRange) {
                probs = output;
            }
        }

        auto maxIt = std::max_element(probs.begin(), probs.end());
        int idx = static_cast<int>(std::distance(probs.begin(), maxIt));

        // Map index → label; if model has >3 classes, clamp to known set.
        if (idx >= 0 && idx < static_cast<int>(m_labels.size())) {
            result.label = m_labels[static_cast<size_t>(idx)];
        } else if (idx >= static_cast<int>(m_labels.size())) {
            result.label = "attack"; // treat higher class ids as hostile
        } else {
            result.label = "normal";
        }
        result.confidence = *maxIt;
    }

    result.confidence = std::clamp(result.confidence, 0.0f, 1.0f);
    return result;
}

PredictionResult ONNXInference::processInt64Labels(const Ort::Value& outputTensor) {
    auto info = outputTensor.GetTensorTypeAndShapeInfo();
    size_t count = info.GetElementCount();
    if (count == 0) {
        return fallbackResult();
    }

    const int64_t* data = outputTensor.GetTensorData<int64_t>();
    int64_t labelIdx = data[0];

    PredictionResult result;
    if (labelIdx >= 0 && labelIdx < static_cast<int64_t>(m_labels.size())) {
        result.label = m_labels[static_cast<size_t>(labelIdx)];
    } else if (labelIdx >= static_cast<int64_t>(m_labels.size())) {
        result.label = "attack";
    } else {
        result.label = "normal";
    }
    // Label-only outputs have no calibrated probability.
    result.confidence = 0.75f;
    return result;
}
