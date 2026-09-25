# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Go honeypot API (decoy surface + feature extraction + decision engine)
# -----------------------------------------------------------------------------
FROM golang:1.26-bookworm AS honeypot-build

WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc g++ libc6-dev \
    && rm -rf /var/lib/apt/lists/*

COPY honey_pot/go.mod honey_pot/go.sum ./
RUN go mod download

COPY honey_pot/ ./
# CGO required for github.com/mattn/go-sqlite3
ENV CGO_ENABLED=1
RUN go build -o /out/honeypot .

FROM debian:bookworm-slim AS honeypot
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/logs
WORKDIR /app
COPY --from=honeypot-build /out/honeypot /app/honeypot
ENV HONEYPOT_HOST=0.0.0.0 \
    HONEYPOT_PORT=8080 \
    ML_SERVICE_URL=http://ml-service:5000/predict \
    LOG_DB_PATH=/app/logs/honeypot.db \
    LOG_JSON_PATH=/app/logs/requests.json
EXPOSE 8080
VOLUME ["/app/logs"]
CMD ["/app/honeypot"]
