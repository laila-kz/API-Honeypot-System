# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# C++ ONNX ML inference service
# -----------------------------------------------------------------------------
FROM ubuntu:22.04 AS build

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    cmake \
    git \
    curl \
    wget \
    ca-certificates \
    pkg-config \
    nlohmann-json3-dev \
    libspdlog-dev \
    && rm -rf /var/lib/apt/lists/*

# ONNX Runtime (prebuilt)
ARG ORT_VERSION=1.16.3
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/onnxruntime-linux-x64-${ORT_VERSION}.tgz \
    && tar -xzf onnxruntime-linux-x64-${ORT_VERSION}.tgz \
    && cp -r onnxruntime-linux-x64-${ORT_VERSION}/include/* /usr/local/include/ \
    && cp -r onnxruntime-linux-x64-${ORT_VERSION}/lib/* /usr/local/lib/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-${ORT_VERSION}*

# Minimal Findonnxruntime.cmake for non-vcpkg builds
RUN mkdir -p /usr/local/share/cmake/onnxruntime && printf '%s\n' \
    'if(NOT TARGET onnxruntime::onnxruntime)' \
    '  find_path(ONNXRUNTIME_INCLUDE_DIR onnxruntime_cxx_api.h)' \
    '  find_library(ONNXRUNTIME_LIBRARY onnxruntime)' \
    '  add_library(onnxruntime::onnxruntime UNKNOWN IMPORTED)' \
    '  set_target_properties(onnxruntime::onnxruntime PROPERTIES' \
    '    IMPORTED_LOCATION "${ONNXRUNTIME_LIBRARY}"' \
    '    INTERFACE_INCLUDE_DIRECTORIES "${ONNXRUNTIME_INCLUDE_DIR}")' \
    'endif()' \
    'set(onnxruntime_FOUND TRUE)' \
    > /usr/local/share/cmake/onnxruntime/onnxruntimeConfig.cmake

WORKDIR /src
COPY ml-service/CMakeLists.txt ./
COPY ml-service/src ./src
RUN mkdir -p models

RUN cmake -B build -S . \
      -DCMAKE_BUILD_TYPE=Release \
      -DCMAKE_PREFIX_PATH=/usr/local \
      -DML_SERVICE_FETCH_HTTPLIB=ON \
    && cmake --build build --config Release -j"$(nproc)"

FROM ubuntu:22.04 AS runtime
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/models

ARG ORT_VERSION=1.16.3
COPY --from=build /usr/local/lib/libonnxruntime* /usr/local/lib/
RUN ldconfig

WORKDIR /app
COPY --from=build /src/build/bin/ml_service_cpp /app/ml_service_cpp
# Model must be mounted or copied at deploy time
ENV ML_SERVICE_PORT=5000 \
    ONNX_MODEL_PATH=/app/models/random_forest.onnx
EXPOSE 5000
VOLUME ["/app/models"]
CMD ["/app/ml_service_cpp"]
