# Mosquitto Exporter

[![CI](https://github.com/d0ugal/mosquitto-exporter/workflows/CI/badge.svg)](https://github.com/d0ugal/mosquitto-exporter/actions)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

Prometheus exporter for [Mosquitto MQTT broker](https://mosquitto.org/) metrics, providing real-time monitoring of broker statistics through the `$SYS/#` topic hierarchy.

## About This Fork

This is a modernized version of the [original mosquitto-exporter](https://github.com/sapcc/mosquitto-exporter) created by Arturo Reuschenbach Puncernau and Fabian Ruff at SAP Converged Cloud. The original project was built with Go 1.14 and has been completely refactored with:

- **Modern Go 1.24+** with updated dependencies
- **Prometheus exporter framework** using [promexporter](https://github.com/d0ugal/promexporter)
- **Structured logging** with slog (JSON/text formats)
- **Web UI dashboard** showing exporter status and configuration
- **Health endpoint** for container orchestration
- **Optional observability** features (OpenTelemetry tracing, Pyroscope profiling)
- **YAML configuration** files with environment variable support
- **Automated releases** with semantic versioning

All original functionality has been preserved while adding modern DevOps practices and observability features.

### Original Authors

**Credit to the original creators:**
- Arturo Reuschenbach Puncernau ([@areusch](https://github.com/areusch)) - SAP Converged Cloud
- Fabian Ruff ([@fruff](https://github.com/fruff)) - SAP Converged Cloud

This project respects the original Apache 2.0 license and builds upon their excellent foundation.

---

## Features

- **Real-time Metrics**: Subscribes to Mosquitto's `$SYS/#` topics for live broker statistics
- **Prometheus Compatible**: Exposes metrics in Prometheus/OpenMetrics format
- **Counter & Gauge Metrics**: Automatically detects metric types (bytes sent/received, client counts, message rates, etc.)
- **Web UI**: User-friendly dashboard at `/` showing exporter info and configuration
- **Health Checks**: `/health` endpoint for Kubernetes liveness/readiness probes
- **Secure by Default**: Supports TLS/SSL connections and MQTT authentication
- **Connection Resilience**: Automatic reconnection with exponential backoff
- **Structured Logging**: JSON or text format logs with configurable levels
- **Configuration**: YAML files or environment variables
- **Optional Tracing**: OpenTelemetry integration for distributed tracing
- **Optional Profiling**: Pyroscope integration for continuous profiling

## Quick Start

### Docker (Recommended)

```bash
docker run -p 9234:9234 \
  ghcr.io/d0ugal/mosquitto-exporter:latest \
  --config-from-env \
  -e MOSQUITTO_BROKER_ENDPOINT=tcp://mosquitto:1883
```

### Docker Compose

```yaml
version: '3.8'
services:
  mosquitto-exporter:
    image: ghcr.io/d0ugal/mosquitto-exporter:latest
    ports:
      - "9234:9234"
    environment:
      MOSQUITTO_BROKER_ENDPOINT: tcp://mosquitto:1883
      MOSQUITTO_USERNAME: ""
      MOSQUITTO_PASSWORD: ""
      LOG_LEVEL: info
      LOG_FORMAT: json
    restart: unless-stopped
```

### Binary

```bash
# Download latest release
wget https://github.com/d0ugal/mosquitto-exporter/releases/latest/download/mosquitto-exporter

# Run with environment variables
MOSQUITTO_BROKER_ENDPOINT=tcp://localhost:1883 ./mosquitto-exporter --config-from-env

# Or with config file
./mosquitto-exporter --config config.yaml
```

## Configuration

### Configuration File (config.yaml)

```yaml
# Server configuration
server:
  host: "0.0.0.0"
  port: 9234
  enable_web_ui: true
  enable_health: true

# Logging configuration
logging:
  level: "info"        # debug, info, warn, error
  format: "json"       # json, text

# Mosquitto broker configuration
mosquitto:
  broker_endpoint: "tcp://127.0.0.1:1883"
  username: ""
  password: ""
  client_id: ""
  
  # TLS/SSL configuration
  tls:
    enabled: false
    cert_file: "/path/to/cert.pem"
    key_file: "/path/to/key.pem"
    insecure_skip_verify: false

# Optional: OpenTelemetry tracing
tracing:
  enabled: false
  service_name: "mosquitto-exporter"
  endpoint: "http://localhost:4318/v1/traces"

# Optional: Pyroscope profiling
profiling:
  enabled: false
  service_name: "mosquitto-exporter"
  server_address: "http://localhost:4040"
```

See [config.yaml.example](config.yaml.example) for a complete configuration example.

### Environment Variables

#### New Variable Names (Recommended)

| Variable | Description | Default |
|----------|-------------|---------|
| `MOSQUITTO_BROKER_ENDPOINT` | MQTT broker endpoint | `tcp://127.0.0.1:1883` |
| `MOSQUITTO_USERNAME` | MQTT username | - |
| `MOSQUITTO_PASSWORD` | MQTT password | - |
| `MOSQUITTO_CLIENT_ID` | MQTT client ID | Auto-generated |
| `MOSQUITTO_TLS_CERT_FILE` | TLS certificate path | - |
| `MOSQUITTO_TLS_KEY_FILE` | TLS key path | - |
| `MOSQUITTO_TLS_INSECURE_SKIP_VERIFY` | Skip TLS verification | `false` |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` |
| `SERVER_PORT` | HTTP server port | `9234` |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `LOG_FORMAT` | Log format (json, text) | `json` |
| `TRACING_ENABLED` | Enable OpenTelemetry tracing | `false` |
| `PROFILING_ENABLED` | Enable Pyroscope profiling | `false` |

#### Legacy Variable Names (Still Supported)

For backward compatibility, the following legacy environment variables are supported:

- `BROKER_ENDPOINT` → `MOSQUITTO_BROKER_ENDPOINT`
- `MQTT_USER` → `MOSQUITTO_USERNAME`
- `MQTT_PASS` → `MOSQUITTO_PASSWORD`
- `MQTT_CLIENT_ID` → `MOSQUITTO_CLIENT_ID`
- `MQTT_CERT` → `MOSQUITTO_TLS_CERT_FILE`
- `MQTT_KEY` → `MOSQUITTO_TLS_KEY_FILE`
- `BIND_ADDRESS` → Parsed to `SERVER_HOST` and `SERVER_PORT`

### Command-Line Flags

```bash
mosquitto-exporter [OPTIONS]

Options:
  --config PATH              Path to YAML configuration file (default: config.yaml)
  --config-from-env          Load configuration entirely from environment variables
  --show-config              Display loaded configuration and exit
  --version, -v              Show version information
  --help, -h                 Show help message
```

## Exposed Metrics

The exporter subscribes to the Mosquitto `$SYS/#` topic hierarchy and exposes all broker metrics as Prometheus metrics.

### Metric Types

- **Counters**: Monotonically increasing values (bytes sent/received, message totals, client totals)
- **Gauges**: Point-in-time values (current connections, queue sizes, load averages)

### Example Metrics

```prometheus
# HELP broker_bytes_received Total bytes received by the broker
# TYPE broker_bytes_received counter
broker_bytes_received 1.05844426e+08

# HELP broker_clients_connected Current number of connected clients
# TYPE broker_clients_connected gauge
broker_clients_connected 9

# HELP broker_messages_received Total messages received since broker started
# TYPE broker_messages_received counter
broker_messages_received 2.456789e+06

# HELP broker_uptime Broker uptime in seconds
# TYPE broker_uptime counter
broker_uptime 86400
```

### Endpoints

- **`/`** - Web UI dashboard (if enabled)
- **`/metrics`** - Prometheus metrics endpoint
- **`/health`** - Health check endpoint (returns JSON with status)

## Building from Source

### Prerequisites

- Go 1.24 or later
- Docker (optional, for containerized builds)
- Make

### Build Commands

```bash
# Build binary
make build

# Run tests
make test

# Run linting
make lint

# Generate development tag
make dev-tag

# Clean build artifacts
make clean
```

### Docker Build

```bash
docker build -t mosquitto-exporter:local .
```

## Metrics Comparison with Original

This modernized version maintains **100% compatibility** with the original exporter's metrics:

- ✅ Same metric names (e.g., `broker_bytes_received`, `broker_clients_connected`)
- ✅ Same metric types (counters vs gauges)
- ✅ Same default port (9234)
- ✅ Same MQTT subscription pattern (`$SYS/#`)
- ✅ Same topic filtering and metric transformation logic

**Added features** that don't affect existing metrics:
- Web UI dashboard at `/` (doesn't interfere with `/metrics`)
- Health endpoint at `/health`
- Structured JSON logs (optional, can use text format)
- Additional runtime metrics (Go metrics, process metrics) from Prometheus client

## Migration from Original

If you're upgrading from the original sapcc/mosquitto-exporter:

1. **Docker Image**: Update image reference from `sapcc/mosquitto-exporter` to `ghcr.io/d0ugal/mosquitto-exporter`
2. **Environment Variables**: Old variable names still work, but consider migrating to new names
3. **Configuration**: Optionally create a `config.yaml` file for easier management
4. **CLI Flags**: The old flags are no longer supported, use environment variables or config file
5. **Metrics**: No changes needed - all metric names and types remain identical

### Example Migration

**Before (original):**
```bash
docker run -p 9234:9234 sapcc/mosquitto-exporter \
  --endpoint tcp://mosquitto:1883 \
  --user myuser \
  --pass mypass
```

**After (modernized):**
```bash
docker run -p 9234:9234 ghcr.io/d0ugal/mosquitto-exporter:latest \
  -e MOSQUITTO_BROKER_ENDPOINT=tcp://mosquitto:1883 \
  -e MOSQUITTO_USERNAME=myuser \
  -e MOSQUITTO_PASSWORD=mypass
```

Or using legacy variable names (still supported):
```bash
docker run -p 9234:9234 ghcr.io/d0ugal/mosquitto-exporter:latest \
  -e BROKER_ENDPOINT=tcp://mosquitto:1883 \
  -e MQTT_USER=myuser \
  -e MQTT_PASS=mypass
```

## Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mosquitto-exporter
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mosquitto-exporter
  template:
    metadata:
      labels:
        app: mosquitto-exporter
    spec:
      containers:
      - name: mosquitto-exporter
        image: ghcr.io/d0ugal/mosquitto-exporter:latest
        ports:
        - containerPort: 9234
          name: metrics
        env:
        - name: MOSQUITTO_BROKER_ENDPOINT
          value: "tcp://mosquitto:1883"
        - name: LOG_LEVEL
          value: "info"
        - name: LOG_FORMAT
          value: "json"
        livenessProbe:
          httpGet:
            path: /health
            port: metrics
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: metrics
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: mosquitto-exporter
  labels:
    app: mosquitto-exporter
spec:
  ports:
  - port: 9234
    name: metrics
  selector:
    app: mosquitto-exporter
```

## Prometheus Configuration

Add to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'mosquitto'
    static_configs:
      - targets: ['mosquitto-exporter:9234']
    scrape_interval: 30s
```

## Troubleshooting

### Connection Issues

If the exporter can't connect to Mosquitto:

1. **Check endpoint format**: Must be `tcp://host:port`, `ssl://host:port`, or `tls://host:port`
2. **Verify network connectivity**: Ensure the exporter can reach the broker
3. **Check authentication**: If Mosquitto requires auth, provide username/password
4. **TLS issues**: For TLS connections, ensure certificates are valid and paths are correct

### View Logs

```bash
# Text format logs (human-readable)
LOG_FORMAT=text ./mosquitto-exporter --config-from-env

# Debug level logs
LOG_LEVEL=debug ./mosquitto-exporter --config-from-env
```

### No Metrics Appearing

1. **Check MQTT subscription**: Ensure the broker publishes to `$SYS/#` topics
2. **Verify permissions**: The MQTT user must have read access to `$SYS/#`
3. **Check ignored metrics**: Some metrics are intentionally filtered (see source code)

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes with clear commit messages
4. Run tests and linting (`make test lint`)
5. Push to your branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

This maintains the same license as the original project by Arturo Reuschenbach Puncernau and Fabian Ruff at SAP Converged Cloud.

## Acknowledgments

- **Original Authors**: Arturo Reuschenbach Puncernau and Fabian Ruff (SAP Converged Cloud)
- **Original Repository**: [sapcc/mosquitto-exporter](https://github.com/sapcc/mosquitto-exporter)
- **Mosquitto Project**: [Eclipse Mosquitto](https://mosquitto.org/)
- **Prometheus**: [Prometheus Monitoring System](https://prometheus.io/)

## Links

- [GitHub Repository](https://github.com/d0ugal/mosquitto-exporter)
- [Docker Images](https://ghcr.io/d0ugal/mosquitto-exporter)
- [Original Project](https://github.com/sapcc/mosquitto-exporter)
- [Issue Tracker](https://github.com/d0ugal/mosquitto-exporter/issues)
- [Changelog](https://github.com/d0ugal/mosquitto-exporter/releases)
