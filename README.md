# ccswresp

[English](README_EN.md)

---

**Protocol Translation Proxy: OpenAI Responses API ↔ Chat Completions API**

让 Codex CLI 通过任何 Chat Completions API 后端运行（DeepSeek、OpenAI 等）。

Codex CLI 使用 OpenAI Responses API 协议，但很多 LLM 服务只提供 Chat Completions API。
ccswresp 在本地启动一个协议翻译代理，在两者之间无缝转换。

**零运行时依赖** — 单文件 Go 静态二进制，下载即可运行。

## 快速开始

### 方式 1：Homebrew (macOS)

```bash
brew tap uhozicloud/ccswresp
brew install ccswresp

ccswresp --init
# 编辑 ~/.ccswresp/.env，设置 API key
ccswresp
```

### 方式 2：yum / dnf (RHEL/CentOS/Fedora)

```bash
# 从 GitHub Releases 下载 RPM 包
sudo yum install ./ccswresp-1.0.0-1.x86_64.rpm
# 或
sudo dnf install ./ccswresp-1.0.0-1.x86_64.rpm

ccswresp --init
ccswresp
```

### 方式 3：apt (Debian/Ubuntu)

```bash
# 从 GitHub Releases 下载 DEB 包
sudo apt install ./ccswresp_1.0.0_amd64.deb

ccswresp --init
ccswresp
```

### 方式 4：一键安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/uhozicloud/ccswresp/main/scripts/install.sh | bash
```

### 方式 5：Windows

从 [GitHub Releases](https://github.com/uhozicloud/ccswresp/releases) 下载 `ccswresp_windows-amd64.zip`，解压后将 `ccswresp.exe` 放到 PATH 中。

### 方式 6：从源码编译 (Go)

```bash
git clone https://github.com/uhozicloud/ccswresp.git
cd ccswresp
go build -o /usr/local/bin/ccswresp .
```

## 使用

```bash
# 默认启动 (端口 11435，DeepSeek API)
ccswresp

# 自定义端口和模型
ccswresp -p 8080 -m deepseek-chat

# 使用 OpenAI 兼容后端
ccswresp -u https://api.openai.com/v1 -k sk-xxx -m gpt-4o

# 查看帮助
ccswresp --help
```

启动后，设置 Codex CLI 使用本地代理：

```bash
# Codex CLI 会自动检测并使用 http://127.0.0.1:11435/v1/responses
# 或者手动设置环境变量
export OPENAI_BASE_URL=http://127.0.0.1:11435/v1
```

## CLI 选项

| 选项 | 环境变量 | 默认值 | 说明 |
|------|---------|--------|------|
| `-p, --port` | `port` | `11435` | 监听端口 |
| `-b, --bind` | `bind_addr` | `127.0.0.1` | 绑定地址 |
| `-m, --model` | `model` | `deepseek-v4-pro` | 默认模型名称 |
| `-u, --base-url` | `base_url` | `https://api.deepseek.com` | 上游 API 地址 |
| `-k, --api-key` | `api_key` | - | API 密钥 |
| `-c, --config` | - | - | 配置文件路径 |
| `--init` | - | - | 初始化 ~/.ccswresp/.env |
| `-q, --quiet` | - | - | 静默模式 |
| `-V, --version` | - | - | 显示版本 |
| `-h, --help` | - | - | 显示帮助 |

## 翻译覆盖

### 输入 (Responses → Chat Completions)

- message items (`input_text` / `output_text` / `reasoning_text`)
- `function_call` → assistant `tool_calls`
- `function_call_output` → `tool` message
- `reasoning` items（跳过，保留 `reasoning_content`）
- `developer` role → `system`
- `input_image` → 跳过统计
- `input_file` / `input_audio` → 跳过统计

### 输出 (Chat Completions → Responses SSE)

- `response.created` / `in_progress` / `completed`
- `output_item.added` / `done`
- `output_text.delta` / `done` + `content_part.added` / `done`
- `reasoning_text.delta` / `done` + `content_part.added` / `done`
- `function_call_arguments.delta` / `done`
- `usage` token 统计（`response.completed` 中）

### 请求参数

- `instructions` → system message
- `temperature` / `top_p` / `max_output_tokens` 透传
- `tools` / `tool_choice` 翻译
- `thinking` / `reasoning` → DeepSeek thinking 模式
- `reasoning_content` 跨轮次自动补回

## 支持的 LLM 后端

ccswresp 可以对接任何兼容 OpenAI Chat Completions API 的后端：

- **DeepSeek** (默认) — `deepseek-v4-pro`, `deepseek-chat`
- **OpenAI** — `gpt-4o`, `gpt-4-turbo` 等
- **兼容 OpenAI API 的本地模型** — Ollama, vLLM, LocalAI 等

## 配置优先级

配置文件按以下顺序加载（先找到的生效）：

1. `-c` 指定的配置文件
2. 当前目录的 `.env`
3. `~/.ccswresp/.env`

CLI 参数会覆盖所有配置文件的值。

## 运行测试

```bash
go test -v ./...
```

37 个翻译逻辑单元测试，不依赖网络。

## 工作原理

```
Codex CLI (Responses API) ────► ccswresp (127.0.0.1:11435)
                                      │
                                      │ Protocol Translation
                                      │
                                      ▼
                               DeepSeek / OpenAI / etc
                               (Chat Completions API)
```

1. Codex CLI 发送 Responses API 请求到 ccswresp
2. ccswresp 将请求翻译为 Chat Completions API 格式
3. ccswresp 将上游响应翻译回 Responses API 格式（支持 SSE 流式）
4. Codex CLI 收到标准 Responses API 响应

## 技术栈

- **Go** — 无运行时依赖，单文件静态二进制
- 标准库 `net/http` — HTTP 服务器和反向代理
- 标准库 `encoding/json` — JSON 处理
- 零外部依赖

## License

MIT © [uhouzicloud](https://github.com/uhozicloud)
