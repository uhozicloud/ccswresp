package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const version = "1.0.0"

func main() {
	// Parse CLI flags
	port := flag.Int("p", 0, "Listen port (env: port, default: 11435)")
	bind := flag.String("b", "", "Bind address (env: bind_addr, default: 127.0.0.1)")
	model := flag.String("m", "", "Default model name (env: model, default: deepseek-v4-pro)")
	baseURL := flag.String("u", "", "Upstream Chat Completions API base URL (env: base_url)")
	apiKey := flag.String("k", "", "API key for upstream service (env: api_key)")
	configPath := flag.String("c", "", "Path to .env config file")
	initConfig := flag.Bool("init", false, "Initialize config file at ~/.ccswresp/.env")
	quiet := flag.Bool("q", false, "Suppress request logging")
	showVersion := flag.Bool("V", false, "Show version")
	showHelp := flag.Bool("h", false, "Show help")

	// Support --long-flags
	flag.IntVar(port, "port", 0, "")
	flag.StringVar(bind, "bind", "", "")
	flag.StringVar(model, "model", "", "")
	flag.StringVar(baseURL, "base-url", "", "")
	flag.StringVar(apiKey, "api-key", "", "")
	flag.StringVar(configPath, "config", "", "")
	flag.BoolVar(showVersion, "version", false, "")
	flag.BoolVar(showHelp, "help", false, "")

	flag.Usage = func() {
		printHelp()
	}
	flag.Parse()

	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("ccswresp v%s\n", version)
		os.Exit(0)
	}

	if *initConfig {
		doInitConfig()
		os.Exit(0)
	}

	quietMode = *quiet

	// Handle positional port argument for backwards compat
	if *port == 0 && flag.NArg() > 0 {
		if p, err := strconv.Atoi(flag.Arg(0)); err == nil {
			*port = p
		}
	}

	// Load config from .env files
	loadDotEnv(*configPath)

	// CLI args override env vars
	resolvedPort := 11435
	if *port > 0 {
		resolvedPort = *port
	} else if envPort := os.Getenv("port"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			resolvedPort = p
		}
	}

	resolvedBind := "127.0.0.1"
	if *bind != "" {
		resolvedBind = *bind
	} else if envBind := os.Getenv("bind_addr"); envBind != "" {
		resolvedBind = envBind
	}

	resolvedModel := "deepseek-v4-pro"
	if *model != "" {
		resolvedModel = *model
	} else if envModel := os.Getenv("model"); envModel != "" {
		resolvedModel = envModel
	}

	resolvedBaseURL := "https://api.deepseek.com"
	if *baseURL != "" {
		resolvedBaseURL = *baseURL
	} else if envURL := os.Getenv("base_url"); envURL != "" {
		resolvedBaseURL = envURL
	}

	resolvedAPIKey := ""
	if *apiKey != "" {
		resolvedAPIKey = *apiKey
	} else if envKey := os.Getenv("api_key"); envKey != "" {
		resolvedAPIKey = envKey
	}

	// Create and start server
	server := NewServer(resolvedPort, resolvedBind, resolvedAPIKey, resolvedBaseURL, resolvedModel)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		server.Shutdown()
		os.Exit(0)
	}()

	if err := server.Start(); err != nil {
		if err.Error() != "http: Server closed" {
			fmt.Fprintf(os.Stderr, "Failed to start: %v\n", err)
			if strings.Contains(err.Error(), "address already in use") {
				fmt.Fprintf(os.Stderr, "Port %d is already in use. Try: ccswresp -p %d\n", resolvedPort, resolvedPort+1)
			}
			os.Exit(1)
		}
	}
}

func printHelp() {
	fmt.Printf(`
  %s v%s

  %s

  %s
    ccswresp [options]

  %s
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

  %s
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

  %s
    api_key       API key for the upstream service
    base_url      Upstream Chat Completions API base URL
    model         Default model name
    port          Listen port
    bind_addr     Bind address

  %s
    Priority: .env (cwd) > ~/.ccswresp/.env
    Run 'ccswresp --init' to create ~/.ccswresp/.env

  %s
    GitHub:  %s
    Issues:  %s
`,
		bold("ccswresp"), version,
		cyan("Protocol Translation Proxy: OpenAI Responses API ↔ Chat Completions API"),
		bold("USAGE:"),
		bold("OPTIONS:"),
		bold("EXAMPLES:"),
		bold("ENVIRONMENT:"),
		bold("CONFIG FILE:"),
		bold("LINKS:"),
		cyan("https://github.com/uhozicloud/ccswresp"),
		cyan("https://github.com/uhozicloud/ccswresp/issues"),
	)
}

// loadDotEnv reads .env files in priority order.
func loadDotEnv(explicitPath string) {
	var paths []string

	if explicitPath != "" {
		paths = append(paths, explicitPath)
	}

	// cwd .env
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, ".env"))
	}

	// ~/.ccswresp/.env
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".ccswresp", ".env"))
	}

	for _, p := range paths {
		if parseDotEnv(p) {
			return // First found wins
		}
	}
}

// parseDotEnv parses a simple .env file (KEY=VALUE format).
func parseDotEnv(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Don't override env vars already set
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	return true
}

// doInitConfig creates the default config file at ~/.ccswresp/.env.
func doInitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".ccswresp")
	configFile := filepath.Join(configDir, ".env")

	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Config already exists at %s\n", configFile)
		fmt.Println("Remove it first to re-initialize.")
		return
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create config directory: %v\n", err)
		os.Exit(1)
	}

	template := `# ccswresp Configuration
# See https://github.com/uhozicloud/ccswresp for docs

api_key=sk-your-api-key-here
base_url=https://api.deepseek.com
model=deepseek-v4-pro
port=11435
`
	if err := os.WriteFile(configFile, []byte(template), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config created at %s\n", configFile)
	fmt.Println("Edit this file to set your API key and preferences.")
}
