# API Honeypot System (Go + C++ ONNX)

Production-style API honeypot for security research. The system exposes realistic decoy REST endpoints, extracts request features, calls a C++ ONNX inference service, and returns deceptive responses based on predicted threat level.

## Highlights

- Decoy API surface for attacker interaction and profiling
- ML-assisted classification: normal, suspicious, attack
- Deception-aware responses (fake errors, delays, misleading headers)
- Dual logging pipeline: SQLite + JSON lines
- Separation of concerns: Go traffic layer + C++ low-latency inference service
- Safe fallback behavior when ML service is unavailable

## Repository Structure

```text
honey-pot-project/
├─ honey_pot/                 # Go honeypot API server
│  ├─ main.go
│  ├─ go.mod
│  ├─ client/                 # ML HTTP client
│  ├─ config/                 # Runtime configuration
│  ├─ decisions/              # Response/deception engine
│  ├─ feature/                # Feature extraction
│  ├─ handlers/               # Endpoint handlers
│  ├─ logger/                 # SQLite + JSON logging
│  └─ models/                 # Shared request/response types
├─ ml-service/                # C++ ONNX inference service
│  ├─ CMakeLists.txt
│  ├─ src/
│  └─ models/                 # random_forest.onnx (ignored by git)
├─ logs/                      # Root logs directory (runtime)
└─ .gitignore
```

## System Architecture

```text
Attacker/Scanner
      |
      v
Go Honeypot API (Port 8080)
  - fake endpoints
  - feature extraction
  - request logging
      |
      v
C++ ML Service (Port 5000)
  - ONNX Runtime inference
  - /predict + /health
      |
      v
Decision Engine
  - normal/suspicious/attack response strategy
  - adaptive delays + deception headers
```

## Exposed API Endpoints (Go)

- POST/GET /login
- GET/POST /admin
- GET /api/v1/users
- POST /api/v1/auth
- POST /reset-password
- GET /dashboard
- GET /logs?ip=x.x.x.x

Unknown routes are handled with a decoy 404-style JSON response.

## ML Service Endpoints (C++)

- POST /predict
- GET /health
- GET /

## Prerequisites (Windows)

- Visual Studio 2022 (Desktop development with C++)
- CMake 3.20+
- vcpkg
- Go (version compatible with honey_pot/go.mod)
- Git

Optional for local model generation/testing:

- Python 3.10+
- numpy, scikit-learn, skl2onnx, onnx

## 1) Install C++ Dependencies with vcpkg

```powershell
cd C:\
git clone https://github.com/microsoft/vcpkg.git
cd vcpkg
.\bootstrap-vcpkg.bat
.\vcpkg integrate install

.\vcpkg install onnxruntime:x64-windows
.\vcpkg install nlohmann-json:x64-windows
.\vcpkg install spdlog:x64-windows
```

## 2) Build the C++ ML Service

```powershell
cd C:\Users\kheza\Desktop\honey-pot-project\ml-service

cmake -B build -S . `
  -DCMAKE_TOOLCHAIN_FILE=C:/vcpkg/scripts/buildsystems/vcpkg.cmake `
  -DCMAKE_BUILD_TYPE=Release

cmake --build build --config Release
```

Expected binary:

- ml-service/build/bin/Release/ml_service_cpp.exe

## 3) Provide an ONNX Model

The C++ service expects:

- ml-service/models/random_forest.onnx

You can train/export your model in a separate repository and copy the ONNX artifact here.

Recommended training repo:

- https://github.com/laila-kz/Network-Intrusion-Detection-Model

Or override model path at runtime:

```powershell
$env:ONNX_MODEL_PATH = "C:\path\to\your\model.onnx"
```

## 4) Build the Go Honeypot

```powershell
cd C:\Users\kheza\Desktop\honey-pot-project\honey_pot
go mod tidy
go build -o honeypot.exe
```

## 5) Run the System

Use two terminals.

Terminal A (ML service):

```powershell
cd C:\Users\kheza\Desktop\honey-pot-project\ml-service
.\build\bin\Release\ml_service_cpp.exe
```

Terminal B (Go honeypot):

```powershell
cd C:\Users\kheza\Desktop\honey-pot-project\honey_pot
.\honeypot.exe
```

Default ports:

- Go honeypot: 8080
- C++ ML service: 5000

Optional ML service port override:

```powershell
$env:ML_SERVICE_PORT = "5000"
```

## Quick Verification

```powershell
# ML service health
curl http://localhost:5000/health

# Normal traffic simulation
curl http://localhost:8080/api/v1/users

# Attack-ish simulation
curl -X POST http://localhost:8080/login -H "User-Agent: sqlmap" -d "{\"test\":1}"

# Bursty suspicious traffic
1..10 | ForEach-Object { curl http://localhost:8080/admin }
```

## Logging and Data Retention

Generated at runtime by the Go service:

- honey_pot/logs/honeypot.db (SQLite)
- honey_pot/logs/requests.json (JSON lines)

Example queries:

```powershell
# SQLite (if sqlite3 is installed)
sqlite3 honey_pot/logs/honeypot.db "SELECT ip, endpoint, ml_label, confidence FROM requests ORDER BY timestamp DESC LIMIT 20;"

# PowerShell JSON filtering
Get-Content honey_pot/logs/requests.json |
  ConvertFrom-Json |
  Where-Object { $_.ml_label -eq "attack" } |
  Select-Object timestamp, ip, endpoint, confidence
```

## Threat Response Strategy

- normal:
  - realistic success-like JSON responses
  - short random latency
- suspicious:
  - still plausible responses
  - medium delay + misleading diagnostics
- attack:
  - fake backend/SQL errors and denial patterns
  - longer delays to waste scanner time
  - deceptive headers (for attacker misdirection)

## Security Notes

- This is a research honeypot, not a production authentication service
- Responses are intentionally fabricated
- Sensitive headers are redacted in logs where applicable
- Keep deployment isolated (VM/container/segmented network)
- Do not expose directly to critical infrastructure

## Troubleshooting

### ML service fails to start

```powershell
# Check port usage
netstat -ano | findstr :5000

# Validate model file
Get-ChildItem C:\Users\kheza\Desktop\honey-pot-project\ml-service\models\random_forest.onnx
```

### Go service cannot reach ML service

- Confirm ML service is running
- Confirm ML URL in Go config points to http://localhost:5000/predict
- The Go server falls back to label=normal, confidence=0.5 if ML is unavailable

### ONNX prediction crashes or resets

Potential cause: model output tensor shape differs from parsing assumptions in the C++ inference pipeline.

Actions:

- Add output-shape debug logs in ml-service/src/ONNXInference.cpp
- Normalize output handling for class-probability tensors
- Rebuild and re-run the C++ service

## Performance (Target Characteristics)

Performance varies by hardware and model size. Typical intended profile:

- C++ ONNX inference: low-latency per request
- Go API handling: high throughput for decoy traffic
- End-to-end latency: dominated by intentional deception delays

## Development Roadmap

- Add richer deception templates per endpoint
- Add IP/session-level behavior scoring
- Add dashboard/visualization layer
- Add pluggable threat-intel enrichments
- Expand model feature set and calibration pipeline

## Contributing

Contributions are welcome for research and educational improvements:

1. Fork and create a feature branch
2. Add tests or reproducible validation steps
3. Submit a PR with threat model and rationale

