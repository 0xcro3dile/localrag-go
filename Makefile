.PHONY: build run clean docker setup pdf-service

# Build the binary
build:
	CGO_ENABLED=1 go build -o localrag ./cmd/localrag

# Build optimized binary (smaller, stripped)
build-release:
	CGO_ENABLED=1 go build -ldflags="-w -s" -o localrag ./cmd/localrag

# Setup Python virtual environment
setup:
	python3 -m venv .venv
	.venv/bin/pip install -r python/requirements.txt

# Start PDF service (run in separate terminal)
pdf-service:
	.venv/bin/python python/pdf_service.py

# Run locally
run: build
	./localrag --port 8080 --docs ./documents

# Clean build artifacts
clean:
	rm -f localrag

# Build Docker image
docker:
	docker build -t localrag:latest .

# Run in Docker
docker-run:
	docker run -p 8080:8080 -v $(PWD)/documents:/app/documents localrag:latest

# Run tests
test:
	go test ./...

# Tidy dependencies
tidy:
	go mod tidy

