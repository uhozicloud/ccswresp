# ccswresp

[中文](README.md)

---

**Protocol Translation Proxy: OpenAI Responses API ↔ Chat Completions API**

Run Codex CLI with any Chat Completions API backend (DeepSeek, OpenAI, etc.).

Codex CLI uses the OpenAI Responses API protocol, but many LLM services only
provide a Chat Completions API. ccswresp starts a local protocol translation
proxy that seamlessly converts between the two.

**Zero runtime dependencies** — single static Go binary, download and run.

## Quick Start

### Option 1: Homebrew (macOS)

```bash
brew tap uhozicloud/ccswresp
brew install ccswresp

ccswresp --init
# Edit ~/.ccswresp/.env and set your API key
ccswresp
```

### Option 2: yum / dnf (RHEL/CentOS/Fedora)

```bash
# Download RPM from GitHub Releases
sudo yum install ./ccswresp-1.0.0-1.x86_64.rpm
# or
sudo dnf install ./ccswresp-1.0.0-1.x86_64.rpm

ccswresp --init
ccswresp
```

### Option 3: apt (Debian/Ubuntu)

```bash
# Download DEB from GitHub Releases
sudo apt install ./ccswresp_1.0.0_amd64.deb

ccswresp --init
ccswresp
```

### Option 4: One-liner install script

```bash
curl -fsSL https://raw.githubusercontent.com/uhozicloud/ccswresp/main/scripts/install.sh | bash
```

### Option 5: Windows

Download `ccswresp_windows-amd64.zip` from [GitHub Releases](https://github.com/uhozicloud/ccswresp/releases), extract `ccswresp.exe` and place it in your PATH.

### Option 6: Build from source (Go)

```bash
git clone https://github.com/uhozicloud/ccswresp.git
cd ccswresp
go build -o /usr/local/bin/ccswresp .
```

## Usage

```bash
# Start with defaults (port 11435, DeepSeek API)
ccswresp

# Custom port and model
ccswresp -p 8080 -m deepseek-chat

# Use with OpenAI-compatible endpoint
ccswresp -u https://api.openai.com/v1 -k sk-xxx -m gpt-4o

# Show help
ccswresp --help
```

After starting, configure Codex CLI to use the local proxy:

```bash
# Codex CLI auto-detects http://127.0.0.1:11435/responses
# Or set manually:
export OPENAI_BASE_URL=http://127.0.0.1:11435/v1
```

## CLI Options

| Option | Env Var | Default | Description |
|--------|---------|---------|-------------|
| `-p, --port` | `port` | `11435` | Listen port |
| `-b, --bind` | `bind_addr` | `127.0.0.1` | Bind address |
| `-m, --model` | `model` | `deepseek-v4-pro` | Default model name |
| `-u, --base-url` | `base_url` | `https://api.deepseek.com` | Upstream API base URL |
| `-k, --api-key` | `api_key` | - | API key |
| `-c, --config` | - | - | Path to .env config file |
| `--init` | - | - | Initialize ~/.ccswresp/.env |
| `-q, --quiet` | - | - | Suppress request logging |
| `-V, --version` | - | - | Show version |
| `-h, --help` | - | - | Show help |

## Translation Coverage

### Input (Responses → Chat Completions)

- message items (`input_text` / `output_text` / `reasoning_text`)
- `function_call` → assistant `tool_calls`
- `function_call_output` → `tool` message
- `reasoning` items (skipped, `reasoning_content` preserved)
- `developer` role → `system`
- `input_image` → tracked/skipped
- `input_file` / `input_audio` → tracked/skipped

### Output (Chat Completions → Responses SSE)

- `response.created` / `in_progress` / `completed`
- `output_item.added` / `done`
- `output_text.delta` / `done` + `content_part.added` / `done`
- `reasoning_text.delta` / `done` + `content_part.added` / `done`
- `function_call_arguments.delta` / `done`
- `usage` token stats (in `response.completed`)

### Request Parameters

- `instructions` → system message
- `temperature` / `top_p` / `max_output_tokens` passed through
- `tools` / `tool_choice` translated
- `thinking` / `reasoning` → DeepSeek thinking mode
- `reasoning_content` auto-recovered across turns

## Supported LLM Backends

ccswresp works with any OpenAI Chat Completions API-compatible backend:

- **DeepSeek** (default) — `deepseek-v4-pro`, `deepseek-chat`
- **OpenAI** — `gpt-4o`, `gpt-4-turbo`, etc.
- **OpenAI-compatible local models** — Ollama, vLLM, LocalAI, etc.

## Config Priority

Config files are loaded in this order (first found wins):

1. Path specified with `-c, --config`
2. `.env` in current directory
3. `~/.ccswresp/.env`

CLI arguments override all config file values.

## Running Tests

```bash
go test -v ./...
```

37 unit tests for translation logic. No network access required.

## How It Works

```
Codex CLI (Responses API) ────► ccswresp (127.0.0.1:11435)
                                      │
                                      │ Protocol Translation
                                      │
                                      ▼
                               DeepSeek / OpenAI / etc
                               (Chat Completions API)
```

1. Codex CLI sends Responses API requests to ccswresp
2. ccswresp translates requests to Chat Completions API format
3. ccswresp translates upstream responses back to Responses API format (SSE streaming)
4. Codex CLI receives standard Responses API responses

## Tech Stack

- **Go** — zero runtime dependencies, single static binary
- `net/http` — HTTP server and reverse proxy
- `encoding/json` — JSON handling
- Zero external dependencies

## License

MIT © [uhouzicloud](https://github.com/uhozicloud)
