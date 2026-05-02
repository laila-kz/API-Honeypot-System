FROM ubuntu:22.04

# =========================
# Install dependencies
# =========================
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    git \
    curl \
    wget \
    libcurl4-openssl-dev \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# =========================
# Install ONNX Runtime
# =========================
RUN wget https://github.com/microsoft/onnxruntime/releases/download/v1.16.3/onnxruntime-linux-x64-1.16.3.tgz \
    && tar -xzf onnxruntime-linux-x64-1.16.3.tgz \
    && cp -r onnxruntime-linux-x64-1.16.3/include/* /usr/local/include/ \
    && cp -r onnxruntime-linux-x64-1.16.3/lib/* /usr/local/lib/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-1.16.3*

# =========================
# Set work directory
# =========================
WORKDIR /app

# Copy project
COPY . .

# =========================
# Build C++ service
# =========================
RUN mkdir build && cd build && \
    cmake .. && \
    make -j$(nproc)

# =========================
# Runtime
# =========================
EXPOSE 5000

CMD ["./build/ml_service_cpp"]