# Zero Delta Screener

A high-performance cryptocurrency market screener built with Go, designed to monitor and analyze order books across multiple exchanges in real-time.

## Features

- **Multi-Exchange Support**: Currently supports Bybit and Rate exchanges with modular architecture for easy extension
- **Real-Time Order Book Monitoring**: Fetches and processes order book data at configurable intervals
- **Concurrent Processing**: Leverages Go's concurrency for parallel data fetching across multiple symbols
- **Modular Architecture**: Built with dependency injection using Uber's fx framework
- **Docker Support**: Includes Dockerfile and docker-compose for easy deployment
- **Structured Logging**: Comprehensive logging with zap for debugging and monitoring

## Prerequisites

- Go 1.24.1 or higher
- Docker and Docker Compose (optional, for containerized deployment)
- API credentials for supported exchanges (Bybit, Rate)

## Installation

### Clone the Repository

```bash
git clone https://github.com/igefined/zero-delta-screener.git
cd zero-delta-screener
```

### Install Dependencies

```bash
make mod-download
```

## Configuration

1. Copy the example environment file:
```bash
cp .env.example .env
```

2. Edit `.env` and add your exchange API credentials:
```env
# Rate Exchange Configuration
RATE_API_URL=https://api.rate.com
RATE_API_KEY=your_rate_api_key_here

# Bybit Exchange Configuration
BYBIT_API_URL=https://api.bybit.com
BYBIT_API_KEY=your_bybit_api_key_here
BYBIT_API_SECRET=your_bybit_api_secret_here
```

## Usage

### Local Development

```bash
# Run the application
make run

# Build the binary
make build

# Run built binary
./bin/screener
```

### Docker Deployment

```bash
# Build Docker image
make docker-build

# Run with Docker
make docker-run

# Run with Docker Compose
make docker-compose-up

# View logs
make docker-compose-logs

# Stop services
make docker-compose-down
```

## Development

### Project Structure

```
zero-delta-screener/
├── internal/
│   ├── config/          # Configuration management
│   ├── domain/          # Domain models and interfaces
│   ├── providers/       # Exchange provider implementations
│   │   ├── bybit/       # Bybit exchange integration
│   │   └── rate/        # Rate exchange integration
│   └── screener/        # Core screening service logic
├── pkg/
│   └── logger/          # Logging utilities
├── main.go              # Application entry point
├── Dockerfile           # Container configuration
├── docker-compose.yml   # Multi-container orchestration
└── Makefile            # Build and development tasks
```

### Available Make Commands

```bash
make help              # Display all available commands
make build            # Build the application binary
make run              # Run the application locally
make test             # Run tests
make test-coverage    # Run tests with coverage report
make fmt              # Format code
make vet              # Run go vet
make clean            # Clean build artifacts
make check            # Run all checks (fmt, vet, lint, test)
```

### Adding a New Exchange Provider

1. Create a new package in `internal/providers/`
2. Implement the `domain.Provider` interface
3. Register the provider module in `main.go`
4. Add configuration in `internal/config/`

Example provider interface:
```go
type Provider interface {
    Name() string
    Connect(ctx context.Context) error
    Disconnect() error
    GetSupportedSymbols() []string
    FetchOrderBook(ctx context.Context, symbol string) (*OrderBook, error)
}
```

## Architecture

The application uses a modular architecture with the following key components:

- **Providers**: Exchange-specific implementations for fetching market data
- **Screener Service**: Core service that orchestrates data collection and processing
- **Domain Models**: Shared data structures (OrderBook, Price levels)
- **Dependency Injection**: Uber's fx framework for clean dependency management

### Screening Process

1. Service connects to all configured providers on startup
2. Screening loop runs at 10-second intervals (configurable)
3. For each cycle:
   - Fetches order books for all supported symbols from each provider
   - Processes data concurrently using goroutines
   - Logs results (extend `processOrderBook` method for custom logic)

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test -v ./internal/screener/...
```