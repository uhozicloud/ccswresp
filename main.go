package main

import (
	"bufio"
	"encoding/json"
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
	configPath := flag.String("c", "", "Path to config.json config file")
	initConfig := flag.Bool("init", false, "Initialize config file at ~/.ccswresp/config.json")
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

	// Load config.json (priority: -c path > cwd > ~/.ccswresp/)
	cfg := loadConfig(*configPath)

	// CLI args override config file
	if *port > 0 {
		cfg.Port = *port
	} else if cfg.Port == 0 {
		cfg.Port = 11435
	}
	if *bind != "" {
		cfg.BindAddr = *bind
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = "127.0.0.1"
	}
	if *model != "" {
		cfg.Model = *model
	}
	if cfg.Model == "" {
		cfg.Model = "deepseek-v4-pro"
	}
	if *baseURL != "" {
		cfg.BaseURL = *baseURL
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	if *apiKey != "" {
		cfg.APIKey = *apiKey
	}

	// Create and start server
	server := NewServer(cfg.Port, cfg.BindAddr, cfg.APIKey, cfg.BaseURL, cfg.Model)

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
				fmt.Fprintf(os.Stderr, "Port %d is already in use. Try: ccswresp -p %d\n", cfg.Port, cfg.Port+1)
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
    -c, --config <path>     Path to config.json config file
    --init                  Initialize config file at ~/.ccswresp/config.json
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
    Priority: config.json (cwd) > ~/.ccswresp/config.json
    Run 'ccswresp --init' to create ~/.ccswresp/config.json

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

// Config represents the JSON configuration file.
type Config struct {
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url"`
	Model    string `json:"model"`
	Port     int    `json:"port"`
	BindAddr string `json:"bind_addr"`
}

// loadConfig reads config.json in priority order: explicit path > cwd > ~/.ccswresp/
func loadConfig(explicitPath string) Config {
	var paths []string

	if explicitPath != "" {
		paths = append(paths, explicitPath)
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "config.json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".ccswresp", "config.json"))
	}

	for _, p := range paths {
		if cfg, ok := parseConfigFile(p); ok {
			return cfg
		}
	}
	return Config{}
}

func parseConfigFile(path string) (Config, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, false
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: invalid config %s: %v\n", path, err)
		return Config{}, false
	}
	return cfg, true
}

// doInitConfig interactively creates ~/.ccswresp/config.json.
func doInitConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot find home directory: %v\n", err)
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".ccswresp")
	configFile := filepath.Join(configDir, "config.json")

	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Config already exists at %s\n", configFile)
		fmt.Print("Overwrite? [y/N]: ")
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)
	cfg := Config{
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-v4-pro",
		Port:    11435,
	}

	fmt.Println()
	fmt.Println(bold("ccswresp setup"))
	fmt.Print("Press Enter to accept defaults.\n\n")

	// API Key (required)
	for cfg.APIKey == "" {
		fmt.Print("API Key: ")
		input, _ := reader.ReadString('\n')
		cfg.APIKey = strings.TrimSpace(input)
		if cfg.APIKey == "" {
			fmt.Println("  API key is required.")
		}
	}

	// Base URL
	fmt.Printf("Base URL [%s]: ", cfg.BaseURL)
	input, _ := reader.ReadString('\n')
	if s := strings.TrimSpace(input); s != "" {
		cfg.BaseURL = s
	}

	// Model
	fmt.Printf("Model [%s]: ", cfg.Model)
	input, _ = reader.ReadString('\n')
	if s := strings.TrimSpace(input); s != "" {
		cfg.Model = s
	}

	// Port
	fmt.Printf("Port [%d]: ", cfg.Port)
	input, _ = reader.ReadString('\n')
	if s := strings.TrimSpace(input); s != "" {
		if p, err := strconv.Atoi(s); err == nil && p > 0 {
			cfg.Port = p
		}
	}

	// Write config
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create config directory: %v\n", err)
		os.Exit(1)
	}

	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Cannot write config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n%s Config saved: %s\n", cGreen+"✓"+cReset, configFile)
	fmt.Printf("Start with: %s\n", cyan("ccswresp"))
}
