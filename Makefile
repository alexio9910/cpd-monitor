.PHONY: tidy build run vet stack-up stack-down stack-logs

# Resuelve dependencias y genera go.sum (necesario la primera vez, y
# cada vez que cambies algun import).
tidy:
	go mod tidy

# Compila el binario del colector en ./bin/cpd-monitor
build: tidy
	go build -o bin/cpd-monitor ./cmd/collector

# Compila y ejecuta directamente (util en desarrollo).
run: tidy
	go run ./cmd/collector -config config.yaml

vet: tidy
	go vet ./...

# Levanta InfluxDB + Grafana en segundo plano.
stack-up:
	docker compose up -d

stack-down:
	docker compose down

stack-logs:
	docker compose logs -f
