package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Server handles HTTP requests for the protocol translation proxy.
type Server struct {
	Port         int
	BindAddr     string
	APIKey       string
	BaseURL      string
	DefaultModel string
	httpServer   *http.Server
}

// NewServer creates a new proxy server instance.
func NewServer(port int, bindAddr, apiKey, baseURL, defaultModel string) *Server {
	return &Server{
		Port:         port,
		BindAddr:     bindAddr,
		APIKey:       apiKey,
		BaseURL:      strings.TrimRight(baseURL, "/"),
		DefaultModel: defaultModel,
	}
}

// Start begins listening and serving HTTP requests.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)
	mux.HandleFunc("/v1/responses", s.handleRequest)
	mux.HandleFunc("/responses", s.handleRequest)
	mux.HandleFunc("/v1", s.handleHealth)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.BindAddr, s.Port),
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 600 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println()
	logOk("ccswresp started")
	logInfo(fmt.Sprintf("http://%s:%d/v1/responses", s.BindAddr, s.Port))
	logInfo("model: " + s.DefaultModel)
	logInfo("upstream: " + s.BaseURL)
	if s.APIKey == "" {
		logWarn("api_key not set — set it in .env or use -k flag")
	}
	fmt.Println()

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "ccswresp",
		"version": "1.0.0",
		"model":   s.DefaultModel,
		"status":  "ok",
		"port":    s.Port,
		"base_url": s.BaseURL,
	})
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	s.setCORS(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}

	if r.Method == "GET" {
		s.handleHealth(w, r)
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"error":{"message":"method not allowed"}}`, http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "failed to read body: "+err.Error())
		return
	}

	var body map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Build upstream Chat Completions request
	chatBody, stream, messages := s.buildChatBody(body)

	// Proxy to upstream
	s.proxyToUpstream(w, chatBody, stream, messages)
}

func (s *Server) buildChatBody(body map[string]interface{}) (map[string]interface{}, bool, []map[string]interface{}) {
	stream := true
	if v, ok := body["stream"].(bool); ok {
		stream = v
	}

	// Determine thinking mode
	enableThinking := false
	if v, ok := body["thinking"].(bool); ok && v {
		enableThinking = true
	}
	if thinking, ok := body["thinking"].(map[string]interface{}); ok {
		if t, _ := thinking["type"].(string); t == "enabled" {
			enableThinking = true
		}
	}
	if reasoning, ok := body["reasoning"].(map[string]interface{}); ok {
		if _, ok := reasoning["effort"]; ok {
			enableThinking = true
		}
	}

	// Translate input messages
	messages, stats := translateInput(body["input"], enableThinking)
	_ = stats // stats are logged below

	// Recover reasoning content
	restored := recoverReasoning(messages)

	// Check message state
	hasAssistantWithRC := false
	hasAssistantWithTC := false
	for _, msg := range messages {
		if role, _ := msg["role"].(string); role == "assistant" {
			if _, ok := msg["reasoning_content"]; ok {
				hasAssistantWithRC = true
			}
			if _, ok := msg["tool_calls"]; ok {
				hasAssistantWithTC = true
			}
		}
	}

	effectiveThinking := enableThinking && (hasAssistantWithRC || !hasAssistantWithTC)

	if enableThinking && !effectiveThinking {
		logWarn("thinking off: missing rc in history")
	}
	if restored > 0 && effectiveThinking {
		logOk(fmt.Sprintf("rc restored x%d", restored))
	}
	if stats.StrippedReasoningContent > 0 {
		logSkip(fmt.Sprintf("rc stripped x%d", stats.StrippedReasoningContent))
	}
	if stats.PreservedReasoningContent > 0 && restored == 0 {
		logInfo(fmt.Sprintf("rc preserved x%d", stats.PreservedReasoningContent))
	}

	// Log request preview
	lastUser := lastUserText(messages)
	preview := previewText(lastUser, 120)
	logReq(fmt.Sprintf("thinking:%v msgs:%d stream:%v | %s",
		effectiveThinking, len(messages), stream, preview))

	// Prepend system instruction
	identity := formatIdentityInstruction()
	instructions := ""
	if v, ok := body["instructions"].(string); ok && v != "" {
		instructions = v + identity
	} else {
		instructions = strings.TrimSpace(identity)
	}

	systemMsg := map[string]interface{}{
		"role":    "system",
		"content": instructions,
	}
	messages = append([]map[string]interface{}{systemMsg}, messages...)

	// Build chat body
	clientModel, _ := body["model"].(string)
	chatBody := map[string]interface{}{
		"model":    resolveModel(clientModel, s.DefaultModel),
		"messages": messages,
		"stream":   stream,
	}

	// Thinking mode
	if effectiveThinking {
		chatBody["thinking"] = map[string]interface{}{"type": "enabled"}
	} else {
		chatBody["thinking"] = map[string]interface{}{"type": "disabled"}
	}

	// Tools
	tools := translateTools(body["tools"])
	if len(tools) > 0 {
		chatBody["tools"] = tools
		tc := translateToolChoice(body["tool_choice"])
		if tc != nil {
			chatBody["tool_choice"] = tc
		}
	}

	// Sampling params
	if v, ok := body["temperature"].(float64); ok {
		chatBody["temperature"] = v
	}
	if v, ok := body["top_p"].(float64); ok {
		chatBody["top_p"] = v
	}
	if v, ok := body["max_output_tokens"].(float64); ok {
		chatBody["max_tokens"] = v
	}

	return chatBody, stream, messages
}

func (s *Server) proxyToUpstream(w http.ResponseWriter, chatBody map[string]interface{}, stream bool, messages []map[string]interface{}) {
	chatJSON, err := json.Marshal(chatBody)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to marshal request: "+err.Error())
		return
	}

	upstreamURL := s.BaseURL + "/chat/completions"
	if _, err := url.Parse(upstreamURL); err != nil {
		s.writeError(w, http.StatusInternalServerError, "invalid upstream URL: "+err.Error())
		return
	}

	// Create upstream request
	upReq, err := http.NewRequest("POST", upstreamURL, bytes.NewReader(chatJSON))
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "failed to create upstream request: "+err.Error())
		return
	}

	upReq.Header.Set("Authorization", "Bearer "+s.APIKey)
	upReq.Header.Set("Content-Type", "application/json")
	if stream {
		upReq.Header.Set("Accept", "text/event-stream")
	} else {
		upReq.Header.Set("Accept", "application/json")
	}

	client := &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}

	resp, err := client.Do(upReq)
	if err != nil {
		s.writeError(w, http.StatusBadGateway, "upstream connection failed: "+err.Error())
		logErr("connect: " + err.Error())
		return
	}
	defer resp.Body.Close()

	// Handle upstream errors
	if resp.StatusCode != 200 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		statusCode := resp.StatusCode
		if statusCode >= 500 {
			statusCode = http.StatusBadGateway
		}
		logErr(fmt.Sprintf("Upstream %d: %s", resp.StatusCode, string(errBody)))
		s.writeError(w, statusCode, fmt.Sprintf("Upstream %d: %s", resp.StatusCode, string(errBody)))
		return
	}

	// Non-streaming response
	if !stream {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			s.writeError(w, http.StatusBadGateway, "failed to read upstream response: "+err.Error())
			return
		}

		var completion map[string]interface{}
		if err := json.Unmarshal(respBody, &completion); err != nil {
			s.writeError(w, http.StatusBadGateway, "failed to parse upstream response: "+err.Error())
			return
		}

		// Remember reasoning for future requests
		if choices, ok := completion["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if msg, ok := choice["message"].(map[string]interface{}); ok {
					rememberReasoning([]map[string]interface{}{msg})
				}
			}
		}

		response := s.buildNonStreamResponse(completion, chatBody["model"].(string))

		// Log token usage
		if usage, ok := completion["usage"].(map[string]interface{}); ok {
			p, _ := usage["prompt_tokens"].(float64)
			c, _ := usage["completion_tokens"].(float64)
			t, _ := usage["total_tokens"].(float64)
			logToks(int(p), int(c), int(t))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Streaming (SSE) response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(200)

	translator := newSseTranslator(w, chatBody["model"].(string))

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonStr := strings.TrimSpace(line[6:])
		if jsonStr == "[DONE]" {
			continue
		}

		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
			continue
		}

		translator.feed(chunk)
	}

	// Remember reasoning content for future requests
	if translator.reasoningSoFar != "" {
		rememberReasoning([]map[string]interface{}{
			{
				"role":              "assistant",
				"content":           translator.contentSoFar,
				"reasoning_content": translator.reasoningSoFar,
			},
		})
	}

	translator.done(nil)
}

func (s *Server) buildNonStreamResponse(completion map[string]interface{}, model string) map[string]interface{} {
	output := []interface{}{}

	choices, _ := completion["choices"].([]interface{})
	if len(choices) > 0 {
		choice, _ := choices[0].(map[string]interface{})
		msg, _ := choice["message"].(map[string]interface{})

		if msg != nil {
			// Reasoning content
			if rc, ok := msg["reasoning_content"].(string); ok && rc != "" {
				output = append(output, map[string]interface{}{
					"id":   randomID("rsn", 6),
					"type": "reasoning",
					"content": []interface{}{
						map[string]interface{}{
							"type": "reasoning_text",
							"text": rc,
						},
					},
					"status": "completed",
				})
			}

			// Text content
			if content, ok := msg["content"].(string); ok && content != "" {
				output = append(output, map[string]interface{}{
					"id":      randomID("msg", 6),
					"type":    "message",
					"role":    "assistant",
					"content": []interface{}{
						map[string]interface{}{
							"type":        "output_text",
							"text":        content,
							"annotations": []interface{}{},
						},
					},
					"status": "completed",
				})
			}

			// Tool calls
			if toolCalls, ok := msg["tool_calls"].([]interface{}); ok {
				for _, tc := range toolCalls {
					tcMap, _ := tc.(map[string]interface{})
					id, _ := tcMap["id"].(string)
					fn, _ := tcMap["function"].(map[string]interface{})
					name, _ := fn["name"].(string)
					args, _ := fn["arguments"].(string)

					output = append(output, map[string]interface{}{
						"id":        "fc_" + id,
						"type":      "function_call",
						"call_id":   id,
						"name":      name,
						"arguments": args,
						"status":    "completed",
					})
				}
			}
		}
	}

	var respUsage interface{}
	if usage, ok := completion["usage"].(map[string]interface{}); ok {
		r := map[string]interface{}{}
		if v, ok := usage["prompt_tokens"].(float64); ok {
			r["input_tokens"] = int(v)
		}
		if v, ok := usage["completion_tokens"].(float64); ok {
			r["output_tokens"] = int(v)
		}
		if v, ok := usage["total_tokens"].(float64); ok {
			r["total_tokens"] = int(v)
		}
		respUsage = r
	}

	return map[string]interface{}{
		"id":     randomID("resp", 8),
		"object": "response",
		"status": "completed",
		"model":  model,
		"output": output,
		"usage":  respUsage,
	}
}

func (s *Server) setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"type":    "proxy_error",
			"code":    fmt.Sprintf("ccswresp_%d", statusCode),
			"message": message,
		},
	})
}
