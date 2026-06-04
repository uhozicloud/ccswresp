# Contributing to ccswresp

Thanks for your interest in contributing!

## Getting Started

```bash
# Requires Go >= 1.21
git clone https://github.com/uhozicloud/ccswresp.git
cd ccswresp

# Build
go build -o ccswresp .

# Run tests
go test -v ./...
```

## Project Structure

```
ccswresp/
├── main.go              # Entry point, CLI argument parsing, config loading
├── server.go            # HTTP server, upstream proxying, request building
├── translate.go         # Protocol translation (Responses → Chat Completions)
├── sse.go               # SSE streaming event translator (Chat → Responses)
├── recover.go           # Reasoning content recovery across turns
├── log.go               # Color terminal logging
├── translate_test.go    # Unit tests for translation logic
├── Makefile             # Build all platforms, packaging
├── scripts/
│   └── install.sh       # Universal install script
└── packaging/
    ├── brew/ccswresp.rb    # Homebrew formula
    ├── rpm/ccswresp.spec   # RPM spec file
    ├── deb/DEBIAN/control   # DEB package control file
    └── nsis/installer.nsi  # Windows NSIS installer
```

## Development

```bash
# Build for current platform
make build

# Run with defaults
./ccswresp

# Run with custom options
./ccswresp -p 8080 -m deepseek-chat

# Run tests
make test

# Build for all platforms
make build-all
```

## Running Tests

```bash
go test -v ./...
```

Tests use Go's built-in testing framework. All translation logic tests are in
`translate_test.go`. No network access is required.

## Pull Requests

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing-thing`)
3. Make your changes
4. Add/update tests
5. Run tests: `go test -v ./...`
6. Commit and push
7. Open a Pull Request

## Coding Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use Go standard library as much as possible (zero external dependencies)
- Add comments for public functions and complex logic
- Keep the codebase simple and readable

## Release Process

1. Update version in `main.go`
2. Run `make test` to ensure all tests pass
3. Run `make dist` to build for all platforms
4. Create a GitHub Release with the binaries from `build/`
5. Update Homebrew formula with new SHA256 hashes
