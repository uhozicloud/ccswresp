# Contributing to ccswresp

Thanks for your interest in contributing!

## Getting Started

```bash
git clone https://github.com/hoganyu/ccswresp.git
cd ccswresp
npm install
```

## Development

```bash
# Run tests
npm test

# Start the proxy (with defaults)
npm start

# Start with custom options
node cli.js -p 8080 -m deepseek-chat
```

## Project Structure

```
ccswresp/
├── cli.js              # CLI entry point (argument parsing, config init)
├── index.js            # HTTP server & main proxy logic
├── lib/
│   ├── log.js          # Color logging utility
│   ├── translate.js    # Protocol translation (Responses → Chat)
│   ├── sse.js          # SSE event translation (Chat → Responses)
│   └── recover.js      # Reasoning content recovery
├── test/
│   └── translate.test.js  # Translation unit tests
├── scripts/
│   ├── install.sh      # Universal install script
│   └── postinstall.js  # npm postinstall hook
└── packaging/
    ├── brew/           # Homebrew formula
    ├── rpm/            # RPM spec file
    └── nsis/           # Windows NSIS installer
```

## Running Tests

```bash
npm test
```

Tests use Node.js built-in test runner. All translation logic tests are in
`test/translate.test.js`. No network access is required.

## Pull Requests

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/amazing-thing`)
3. Make your changes
4. Add/update tests
5. Run tests: `npm test`
6. Commit and push
7. Open a Pull Request

## Coding Style

- Use ES modules (import/export)
- Follow the existing code patterns
- Add JSDoc comments for public APIs
- Keep the dependency footprint small (currently only `dotenv`)
