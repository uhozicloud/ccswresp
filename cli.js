#!/usr/bin/env node
// ccswresp CLI — Command-line interface for the protocol translation proxy

import { startServer } from "./index.js";
import { readFileSync, existsSync, writeFileSync, mkdirSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const PKG = JSON.parse(
  readFileSync(resolve(__dirname, "package.json"), "utf-8")
);

// --- Argument Parsing ---
function parseArgs() {
  const args = process.argv.slice(2);
  const opts = {
    port: null,
    bind: null,
    model: null,
    baseUrl: null,
    apiKey: null,
    config: null,
    init: false,
    help: false,
    version: false,
    quiet: false,
  };

  for (let i = 0; i < args.length; i++) {
    const arg = args[i];
    switch (arg) {
      case "-p":
      case "--port":
        opts.port = parseInt(args[++i], 10);
        break;
      case "-b":
      case "--bind":
        opts.bind = args[++i];
        break;
      case "-m":
      case "--model":
        opts.model = args[++i];
        break;
      case "-u":
      case "--base-url":
        opts.baseUrl = args[++i];
        break;
      case "-k":
      case "--api-key":
        opts.apiKey = args[++i];
        break;
      case "-c":
      case "--config":
        opts.config = args[++i];
        break;
      case "--init":
        opts.init = true;
        break;
      case "-h":
      case "--help":
        opts.help = true;
        break;
      case "-V":
      case "--version":
        opts.version = true;
        break;
      case "-q":
      case "--quiet":
        opts.quiet = true;
        break;
      default:
        // Support positional port argument for backwards compat
        if (!isNaN(parseInt(arg)) && !opts.port) {
          opts.port = parseInt(arg, 10);
        }
    }
  }

  return opts;
}

// --- Help ---
function showHelp() {
  console.log(`
  ${bold("ccswresp")} v${PKG.version}

  ${cyan("Protocol Translation Proxy: OpenAI Responses API ↔ Chat Completions API")}

  ${bold("USAGE:")}
    ccswresp [options]

  ${bold("OPTIONS:")}
    -p, --port <port>       Listen port (default: 11435, env: port)
    -b, --bind <addr>       Bind address (default: 127.0.0.1, env: bind_addr)
    -m, --model <model>     Default model name (default: deepseek-v4-pro, env: model)
    -u, --base-url <url>    Upstream Chat Completions API base URL
                            (default: https://api.deepseek.com, env: base_url)
    -k, --api-key <key>     API key for upstream service (env: api_key)
    -c, --config <path>     Path to .env config file
    --init                  Initialize config file at ~/.ccswresp/.env
    -q, --quiet             Suppress request logging
    -V, --version           Show version
    -h, --help              Show this help

  ${bold("EXAMPLES:")}
    # Start with defaults (DeepSeek)
    ccswresp

    # Start with custom port and model
    ccswresp -p 8080 -m deepseek-chat

    # Use with OpenAI-compatible endpoint
    ccswresp -u https://api.openai.com/v1 -k sk-xxx -m gpt-4o

    # Initialize config file
    ccswresp --init

    # Show version
    ccswresp --version

  ${bold("ENVIRONMENT:")}
    api_key       API key for the upstream service
    base_url      Upstream Chat Completions API base URL
    model         Default model name
    port          Listen port
    bind_addr     Bind address

  ${bold("CONFIG FILE:")}
    Priority: .env (cwd) > ~/.ccswresp/.env > built-in .env
    Run 'ccswresp --init' to create ~/.ccswresp/.env

  ${bold("LINKS:")}
    GitHub:  ${cyan("https://github.com/hoganyu/ccswresp")}
    Issues:  ${cyan("https://github.com/hoganyu/ccswresp/issues")}
`);
}

// --- Init Config ---
function initConfig() {
  const configDir = resolve(
    process.env.HOME || process.env.USERPROFILE || "",
    ".ccswresp"
  );
  const configPath = resolve(configDir, ".env");

  if (existsSync(configPath)) {
    console.log(`Config already exists at ${configPath}`);
    console.log("Remove it first to re-initialize.");
    return;
  }

  mkdirSync(configDir, { recursive: true });

  const template = `# ccswresp Configuration
# See https://github.com/hoganyu/ccswresp for docs

api_key=sk-your-api-key-here
base_url=https://api.deepseek.com
model=deepseek-v4-pro
port=11435
`;
  writeFileSync(configPath, template, "utf-8");
  console.log(`Config created at ${configPath}`);
  console.log("Edit this file to set your API key and preferences.");
}

// --- Terminal Colors ---
function bold(s) {
  return "[1m" + s + "[0m";
}
function cyan(s) {
  return "[36m" + s + "[0m";
}

// --- Main ---
async function main() {
  const opts = parseArgs();

  if (opts.help) {
    showHelp();
    process.exit(0);
  }

  if (opts.version) {
    console.log("ccswresp v" + PKG.version);
    process.exit(0);
  }

  if (opts.init) {
    initConfig();
    process.exit(0);
  }

  // Override env vars with CLI args
  if (opts.port) process.env.port = String(opts.port);
  if (opts.bind) process.env.bind_addr = opts.bind;
  if (opts.model) process.env.model = opts.model;
  if (opts.baseUrl) process.env.base_url = opts.baseUrl;
  if (opts.apiKey) process.env.api_key = opts.apiKey;

  // Re-import to pick up new env values
  const port = parseInt(process.env.port || "11435", 10);
  const bind = process.env.bind_addr || "127.0.0.1";

  try {
    await startServer(port, bind);
  } catch (err) {
    console.error("Failed to start ccswresp: " + err.message);
    if (err.code === "EADDRINUSE") {
      console.error(
        `Port ${port} is already in use. Try: ccswresp -p ${
          port + 1
        }`
      );
    }
    process.exit(1);
  }

  // Graceful shutdown
  const shutdown = () => {
    console.log("\nShutting down...");
    process.exit(0);
  };
  process.on("SIGINT", shutdown);
  process.on("SIGTERM", shutdown);
}

main();
