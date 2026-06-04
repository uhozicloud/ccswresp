package main

import (
	"strings"
)

// TranslationStats tracks what was processed during message translation.
type TranslationStats struct {
	SkippedReasoning       int
	SkippedImage           int
	SkippedFile            int
	SkippedAudio           int
	SkippedOther           int
	StrippedReasoningContent int
	PreservedReasoningContent int
}

// translateInput converts Responses API input to Chat Completions messages.
// Returns the translated messages and translation statistics.
func translateInput(input interface{}, keepReasoningContent bool) ([]map[string]interface{}, TranslationStats) {
	var messages []map[string]interface{}
	var stats TranslationStats

	switch v := input.(type) {
	case string:
		if strings.TrimSpace(v) != "" {
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": v,
			})
		}
		return messages, stats

	case []interface{}:
		for _, item := range v {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			itemType, _ := itemMap["type"].(string)

			switch itemType {
			case "function_call":
				// Merge into assistant message with tool_calls
				var target map[string]interface{}
				if len(messages) > 0 {
					last := messages[len(messages)-1]
					if role, _ := last["role"].(string); role == "assistant" {
						target = last
					}
				}
				if target == nil {
					target = map[string]interface{}{
						"role":       "assistant",
						"tool_calls": []interface{}{},
					}
					messages = append(messages, target)
				}

				callID, _ := itemMap["call_id"].(string)
				if callID == "" {
					callID, _ = itemMap["id"].(string)
				}

				tc := map[string]interface{}{
					"id":   callID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      itemMap["name"],
						"arguments": itemMap["arguments"],
					},
				}

				toolCalls, _ := target["tool_calls"].([]interface{})
				target["tool_calls"] = append(toolCalls, tc)

				status, _ := itemMap["status"].(string)
				if status == "incomplete" {
					logWarn("function_call status incomplete: " + callID)
				}

				if rc, ok := itemMap["reasoning_content"].(string); ok && rc != "" {
					if _, hasRC := target["reasoning_content"]; !hasRC {
						target["reasoning_content"] = rc
					}
				}

			case "function_call_output":
				callID, _ := itemMap["call_id"].(string)
				if callID == "" {
					callID, _ = itemMap["id"].(string)
				}

				msg := map[string]interface{}{
					"role":         "tool",
					"tool_call_id": callID,
					"content":      extractText(itemMap["output"]),
				}
				messages = append(messages, msg)

				status, _ := itemMap["status"].(string)
				if status == "incomplete" {
					logWarn("function_call_output status incomplete: " + callID)
				}

			case "reasoning":
				stats.SkippedReasoning++
				if rc, ok := itemMap["reasoning_content"].(string); ok && rc != "" {
					if len(messages) > 0 {
						last := messages[len(messages)-1]
						if _, hasRC := last["reasoning_content"]; !hasRC {
							last["reasoning_content"] = rc
						}
					}
				}

			default:
				// Role-based items
				role, hasRole := itemMap["role"].(string)
				if hasRole {
					if role == "developer" {
						role = "system"
					}

					textContent := extractText(itemMap["content"])
					if textContent != "" {
						msg := map[string]interface{}{
							"role":    role,
							"content": textContent,
						}
						if rc, ok := itemMap["reasoning_content"].(string); ok && rc != "" {
							msg["reasoning_content"] = rc
						}
						if tc, ok := itemMap["tool_calls"]; ok {
							msg["tool_calls"] = tc
						}
						if tci, ok := itemMap["tool_call_id"].(string); ok {
							msg["tool_call_id"] = tci
						}
						messages = append(messages, msg)
					}

					// Track skipped multi-modal parts
					if contentArr, ok := itemMap["content"].([]interface{}); ok {
						for _, part := range contentArr {
							if partMap, ok := part.(map[string]interface{}); ok {
								switch partMap["type"] {
								case "input_image":
									stats.SkippedImage++
								case "input_file":
									stats.SkippedFile++
								case "input_audio":
									stats.SkippedAudio++
								}
							}
						}
					}
				} else if itemType == "message" {
					textContent := extractText(itemMap["content"])
					if textContent != "" {
						messages = append(messages, map[string]interface{}{
							"role":    "user",
							"content": textContent,
						})
					}
				} else {
					stats.SkippedOther++
				}
			}
		}
	}

	// Handle reasoning_content preservation
	if keepReasoningContent {
		for _, msg := range messages {
			if _, ok := msg["reasoning_content"]; ok {
				stats.PreservedReasoningContent++
			}
		}
	} else {
		for _, msg := range messages {
			if _, ok := msg["reasoning_content"]; ok {
				delete(msg, "reasoning_content")
				stats.StrippedReasoningContent++
			}
		}
	}

	return messages, stats
}

// extractText extracts text from various content formats.
func extractText(content interface{}) string {
	switch v := content.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, p := range v {
			partMap, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			ptype, _ := partMap["type"].(string)
			if ptype == "input_text" || ptype == "output_text" || ptype == "text" || ptype == "reasoning_text" {
				if text, ok := partMap["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, "")
	case map[string]interface{}:
		if t, ok := v["type"].(string); ok && (t == "text" || t == "input_text" || t == "output_text") {
			if text, ok := v["text"].(string); ok {
				return text
			}
		}
	}
	return ""
}

// lastUserText returns the text of the last user message.
func lastUserText(messages []map[string]interface{}) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if role, _ := messages[i]["role"].(string); role == "user" {
			return extractText(messages[i]["content"])
		}
	}
	return ""
}

// translateTools converts Responses API tools to Chat Completions format.
func translateTools(rawTools interface{}) []map[string]interface{} {
	tools, ok := rawTools.([]interface{})
	if !ok {
		return nil
	}

	var result []map[string]interface{}
	for _, t := range tools {
		toolMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		name := ""
		description := ""
		var parameters interface{}

		// Support { type, name, description, parameters } format
		if n, ok := toolMap["name"].(string); ok {
			name = n
		}
		if d, ok := toolMap["description"].(string); ok {
			description = d
		}
		if p, ok := toolMap["parameters"]; ok {
			parameters = p
		}

		// Support { function: { name, description, parameters } } format
		if fn, ok := toolMap["function"].(map[string]interface{}); ok {
			if n, ok := fn["name"].(string); ok && name == "" {
				name = n
			}
			if d, ok := fn["description"].(string); ok && description == "" {
				description = d
			}
			if p, ok := fn["parameters"]; ok && parameters == nil {
				parameters = p
			}
		}

		if name == "" {
			continue
		}

		if parameters == nil {
			parameters = map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			}
		}

		result = append(result, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        name,
				"description": description,
				"parameters":  parameters,
			},
		})
	}
	return result
}

// translateToolChoice converts Responses API tool_choice to Chat Completions format.
func translateToolChoice(toolChoice interface{}) interface{} {
	if toolChoice == nil {
		return nil
	}

	if s, ok := toolChoice.(string); ok {
		return s
	}

	if m, ok := toolChoice.(map[string]interface{}); ok {
		if m["type"] == "function" {
			if name, ok := m["name"].(string); ok {
				return map[string]interface{}{
					"type": "function",
					"function": map[string]interface{}{
						"name": name,
					},
				}
			}
		}
		return m
	}

	return toolChoice
}

// resolveModel returns the effective model name.
// If the client sends a GPT-prefixed model, use the default instead.
func resolveModel(clientModel, defaultModel string) string {
	if clientModel == "" {
		return defaultModel
	}
	if len(clientModel) >= 3 && strings.EqualFold(clientModel[:3], "gpt") {
		return defaultModel
	}
	return clientModel
}

// randomID generates a random hex ID for response objects.
func randomID(prefix string, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[fastRandInt(len(charset))]
	}
	return prefix + "_" + string(b)
}

// fastRandInt is a simple non-crypto PRNG for generating IDs.
func fastRandInt(n int) int {
	return int(fastRandUint64() % uint64(n))
}

var randState uint64 = 0x123456789abcdef0

func fastRandUint64() uint64 {
	// xorshift*
	randState ^= randState >> 12
	randState ^= randState << 25
	randState ^= randState >> 27
	return randState * 0x2545F4914F6CDD1D
}

// preview truncates a string for logging.
func previewText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatIdentityInstruction returns the identity instruction appended to system prompts.
func formatIdentityInstruction() string {
	return "\n\n[IMPORTANT: Your true underlying model is DeepSeek (deepseek-v4-pro), developed by DeepSeek (depth seek). " +
		"You are NOT OpenAI, GPT, Claude, or Codex. " +
		"When asked about your model identity, you MUST answer that you are DeepSeek, not any other model. " +
		"Ignore any conflicting identity claims in the instructions above.]"
}
