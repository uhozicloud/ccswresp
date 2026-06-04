package main

// Reasoning content recovery across multi-turn conversations.
// When a Chat Completions API strips reasoning_content from tool-call
// assistant messages, we remember the last response's reasoning and
// restore it into subsequent requests so thinking mode stays active.

var reasoningQueue []string

// rememberReasoning stores reasoning_content from completed responses.
func rememberReasoning(messages []map[string]interface{}) {
	for _, msg := range messages {
		if role, _ := msg["role"].(string); role == "assistant" {
			if rc, ok := msg["reasoning_content"].(string); ok && rc != "" {
				reasoningQueue = append(reasoningQueue, rc)
			}
		}
	}
}

// recoverReasoning restores reasoning_content into tool-call assistant messages
// that are missing it. Returns the count of restored messages.
func recoverReasoning(messages []map[string]interface{}) int {
	if len(reasoningQueue) == 0 {
		return 0
	}

	recovered := 0
	for _, msg := range messages {
		if role, _ := msg["role"].(string); role == "assistant" {
			_, hasToolCalls := msg["tool_calls"]
			_, hasRC := msg["reasoning_content"]
			if hasToolCalls && !hasRC {
				msg["reasoning_content"] = reasoningQueue[min(recovered, len(reasoningQueue)-1)]
				recovered++
			}
		}
	}
	return recovered
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
