# LocalRAG

A private, offline Retrieval-Augmented Generation (RAG) toolkit built with Go, following Clean Architecture principles.

## Overview

LocalRAG enables you to query your documents using natural language, powered by local LLMs via Ollama. All processing happens on your machine — no data leaves your environment.

## Architecture

This project implements Clean Architecture (also known as Hexagonal Architecture or Ports and Adapters):

```
internal/
├── domain/                 # Core business logic (no external dependencies)
│   ├── entities/           # Document, Chunk, Embedding, QueryResult
│   ├── usecases/           # Ingest, Query business logic
│   └── ports/              # Interface definitions (contracts)
├── adapters/               # Interface implementations
│   ├── embedding/          # Ollama embedding adapter
│   ├── llm/                # Ollama LLM adapter
│   ├── vectordb/           # In-memory and LanceDB stores
│   ├── loader/             # Document loaders (TXT, MD, PDF)
│   └── filewatcher/        # File system monitoring
└── infrastructure/         # Frameworks and drivers
    └── http/               # HTTP server, templates, static files
```

### Design Principles

- **Dependency Inversion**: Use cases depend on port interfaces, not concrete implementations
- **Single Responsibility**: Each component has one reason to change
- **Interface Segregation**: Small, focused interfaces (EmbeddingService, LLMService, VectorStore)
- **Open/Closed**: New adapters can be added without modifying existing code

## Prerequisites

- Go 1.21 or later
- Ollama (https://ollama.com)
- Required models:
  - `nomic-embed-text` (embedding model)
  - `llama3.2` or `tinyllama` (LLM)

## Installation

```bash
# Clone the repository
git clone https://github.com/0xcro3dile/localrag-go.git
cd localrag-go

# Download dependencies
go mod tidy

# Build
go build -o localrag ./cmd/localrag
```

## Usage

### 1. Start Ollama

```bash
ollama serve
```

### 2. Pull Required Models

```bash
ollama pull nomic-embed-text
ollama pull llama3.2
# Or for faster inference on CPU:
ollama pull tinyllama
```

### 3. Run LocalRAG

```bash
./localrag --port 8080 --docs ./documents
```

### 4. Access the Web Interface

Open http://localhost:8080 in your browser.

### 5. Add Documents

Place `.txt` or `.md` files in the `./documents` directory. They will be automatically ingested.

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 8080 | HTTP server port |
| `--docs` | ./documents | Documents directory to watch |
| `--ollama` | http://localhost:11434 | Ollama API URL |
| `--embed-model` | nomic-embed-text | Embedding model name |
| `--llm-model` | llama3.2 | LLM model for generation |

## Docker Deployment

### Build and Run

```bash
docker build -t localrag:latest .
docker run -p 8080:8080 -v $(pwd)/documents:/app/documents localrag:latest
```

### Using Docker Compose (with host Ollama)

```yaml
services:
  localrag:
    build: .
    container_name: localrag
    network_mode: "host"
    volumes:
      - ./documents:/app/documents
    command: ["--port", "8080", "--docs", "/app/documents", "--ollama", "http://127.0.0.1:11434"]
```

```bash
docker-compose up --build
```

## Known Issues and Solutions

### 1. CGO Required for LanceDB

**Problem**: LanceDB uses SQLite which requires CGO. On Windows or in Docker with `CGO_ENABLED=0`, this fails.

**Solution**: The default configuration uses an in-memory vector store which does not require CGO. For persistent storage, build with `CGO_ENABLED=1` and install a C compiler.

### 2. Docker Container Cannot Reach Host Ollama

**Problem**: When running in Docker, the container may not be able to reach Ollama running on the host.

**Solution**: Use `network_mode: "host"` in docker-compose.yml, or configure `extra_hosts` with `host.docker.internal:host-gateway`.

### 3. PDF Support

**Problem**: PDF parsing requires additional libraries.

**Current Status**: A Python-based PDF service exists in `/python/pdf_service.py` but is not connected to the main application. Text and Markdown files work fully.

**Workaround**: Convert PDFs to text files for ingestion.

### 4. Ingestion Hanging

**Problem**: Document ingestion may appear to hang without progress.

**Cause**: The first call to Ollama's embedding API requires loading the model into memory, which can take 30-60 seconds.

**Solution**: Wait for the initial model load. Subsequent ingestions will be faster.

### 5. Port Already in Use

**Problem**: `bind: address already in use` error.

**Solution**: Stop any existing instances or use a different port with `--port 3000`.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Web interface |
| `/api/query` | POST | Query documents (non-streaming) |
| `/api/query/stream` | GET | Query documents (SSE streaming) |
| `/api/health` | GET | Health check |

## Testing

```bash
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

## Project Structure

```
.
├── cmd/localrag/           # Application entry point
├── internal/
│   ├── adapters/           # External service adapters
│   ├── domain/             # Core business logic
│   └── infrastructure/     # HTTP server, templates
├── documents/              # Document storage (gitignored)
├── python/                 # PDF service (optional)
├── Dockerfile
├── docker-compose.yml
├── Makefile
└── README.md
```

## Supported File Types

| Extension | Status |
|-----------|--------|
| `.txt` | Fully supported |
| `.md` | Fully supported |
| `.markdown` | Fully supported |
| `.pdf` | Partial (text extraction only) |

## Performance Considerations

- **Embedding Model**: `nomic-embed-text` provides good quality embeddings at 768 dimensions
- **Chunk Size**: Default 500 characters with 50 character overlap
- **Vector Search**: Top 5 results by cosine similarity
- **Memory Usage**: In-memory store grows with document count

## License

MIT
