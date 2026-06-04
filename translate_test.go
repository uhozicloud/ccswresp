package main

import (
	"testing"
)

// --- extractText tests ---

func TestExtractText_String(t *testing.T) {
	result := extractText("hello")
	if result != "hello" {
		t.Errorf("expected 'hello', got '%s'", result)
	}
}

func TestExtractText_NonArray(t *testing.T) {
	if r := extractText(map[string]interface{}{"a": 1}); r != "" {
		t.Errorf("expected '', got '%s'", r)
	}
	if r := extractText(123); r != "" {
		t.Errorf("expected '', got '%s'", r)
	}
}

func TestExtractText_ContentArray(t *testing.T) {
	result := extractText([]interface{}{
		map[string]interface{}{"type": "input_text", "text": "a"},
		map[string]interface{}{"type": "output_text", "text": "b"},
	})
	if result != "ab" {
		t.Errorf("expected 'ab', got '%s'", result)
	}
}

func TestExtractText_IgnoreNonText(t *testing.T) {
	result := extractText([]interface{}{
		map[string]interface{}{"type": "input_image"},
		map[string]interface{}{"type": "input_text", "text": "t"},
	})
	if result != "t" {
		t.Errorf("expected 't', got '%s'", result)
	}
}

func TestExtractText_SingleObject(t *testing.T) {
	result := extractText(map[string]interface{}{"type": "text", "text": "ok"})
	if result != "ok" {
		t.Errorf("expected 'ok', got '%s'", result)
	}
}

func TestExtractText_ReasoningText(t *testing.T) {
	result := extractText([]interface{}{
		map[string]interface{}{"type": "reasoning_text", "text": "thinking..."},
	})
	if result != "thinking..." {
		t.Errorf("expected 'thinking...', got '%s'", result)
	}
}

// --- translateInput tests ---

func TestTranslateInput_String(t *testing.T) {
	msgs, _ := translateInput("hello", false)
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" {
		t.Errorf("expected role 'user', got '%s'", msgs[0]["role"])
	}
	if msgs[0]["content"] != "hello" {
		t.Errorf("expected content 'hello', got '%s'", msgs[0]["content"])
	}
}

func TestTranslateInput_EmptyString(t *testing.T) {
	msgs, _ := translateInput("   ", false)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestTranslateInput_Null(t *testing.T) {
	msgs, _ := translateInput(nil, false)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestTranslateInput_EmptyArray(t *testing.T) {
	msgs, _ := translateInput([]interface{}{}, false)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestTranslateInput_UserAssistant(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{"type": "input_text", "text": "hi"},
			},
		},
		map[string]interface{}{
			"role": "assistant",
			"content": []interface{}{
				map[string]interface{}{"type": "output_text", "text": "hi!"},
			},
		},
	}, false)

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" || msgs[0]["content"] != "hi" {
		t.Error("first message mismatch")
	}
	if msgs[1]["role"] != "assistant" || msgs[1]["content"] != "hi!" {
		t.Error("second message mismatch")
	}
}

func TestTranslateInput_DeveloperToSystem(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{"role": "developer", "content": "sys"},
	}, false)

	if msgs[0]["role"] != "system" {
		t.Errorf("expected 'system', got '%s'", msgs[0]["role"])
	}
}

func TestTranslateInput_FunctionCallMerge(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{"role": "assistant", "content": []interface{}{}},
		map[string]interface{}{
			"type": "function_call", "call_id": "c1", "name": "f", "arguments": "{}",
		},
		map[string]interface{}{
			"type": "function_call", "call_id": "c2", "name": "g", "arguments": "{}",
		},
	}, false)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["role"] != "assistant" {
		t.Error("expected assistant role")
	}

	toolCalls, ok := msgs[0]["tool_calls"].([]interface{})
	if !ok || len(toolCalls) != 2 {
		t.Fatalf("expected 2 tool_calls, got %v", msgs[0]["tool_calls"])
	}
}

func TestTranslateInput_FunctionCallNoAssistant(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{
			"type": "function_call", "call_id": "c1", "name": "f", "arguments": "{}",
		},
	}, false)

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0]["role"] != "assistant" {
		t.Error("expected assistant role")
	}
}

func TestTranslateInput_FunctionCallOutput(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{
			"type": "function_call_output", "call_id": "c1",
			"output": map[string]interface{}{"type": "text", "text": "ok"},
		},
	}, false)

	if msgs[0]["role"] != "tool" {
		t.Errorf("expected 'tool', got '%s'", msgs[0]["role"])
	}
	if msgs[0]["content"] != "ok" {
		t.Errorf("expected 'ok', got '%s'", msgs[0]["content"])
	}
}

func TestTranslateInput_ReasoningSkipped(t *testing.T) {
	msgs, stats := translateInput([]interface{}{
		map[string]interface{}{"role": "user", "content": "q"},
		map[string]interface{}{"type": "reasoning", "reasoning_content": "t"},
	}, false)

	if len(msgs) != 1 {
		t.Errorf("expected 1 message, got %d", len(msgs))
	}
	if stats.SkippedReasoning != 1 {
		t.Errorf("expected 1 skipped reasoning, got %d", stats.SkippedReasoning)
	}
}

func TestTranslateInput_RCStrippedDefault(t *testing.T) {
	msgs, stats := translateInput([]interface{}{
		map[string]interface{}{"role": "assistant", "content": "a", "reasoning_content": "t"},
	}, false)

	if _, ok := msgs[0]["reasoning_content"]; ok {
		t.Error("reasoning_content should be stripped")
	}
	if stats.StrippedReasoningContent != 1 {
		t.Errorf("expected 1 stripped, got %d", stats.StrippedReasoningContent)
	}
}

func TestTranslateInput_RCKept(t *testing.T) {
	msgs, stats := translateInput([]interface{}{
		map[string]interface{}{"role": "assistant", "content": "a", "reasoning_content": "t"},
	}, true)

	if msgs[0]["reasoning_content"] != "t" {
		t.Error("reasoning_content should be preserved")
	}
	if stats.PreservedReasoningContent != 1 {
		t.Errorf("expected 1 preserved, got %d", stats.PreservedReasoningContent)
	}
}

func TestTranslateInput_MultiModalStats(t *testing.T) {
	_, stats := translateInput([]interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{"type": "input_text", "text": "hi"},
				map[string]interface{}{"type": "input_file"},
				map[string]interface{}{"type": "input_audio"},
			},
		},
	}, false)

	if stats.SkippedFile != 1 {
		t.Errorf("expected 1 skipped file, got %d", stats.SkippedFile)
	}
	if stats.SkippedAudio != 1 {
		t.Errorf("expected 1 skipped audio, got %d", stats.SkippedAudio)
	}
}

func TestTranslateInput_ImageStat(t *testing.T) {
	_, stats := translateInput([]interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{"type": "input_text", "text": "what is this?"},
				map[string]interface{}{"type": "input_image"},
			},
		},
	}, false)

	if stats.SkippedImage != 1 {
		t.Errorf("expected 1 skipped image, got %d", stats.SkippedImage)
	}
}

func TestTranslateInput_FullConversation(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{
			"role": "user", "id": "1",
			"content": []interface{}{
				map[string]interface{}{"type": "input_text", "text": "w?"},
			},
		},
		map[string]interface{}{
			"id": "2", "type": "function_call", "call_id": "abc",
			"name": "f", "arguments": "{}", "status": "completed",
		},
		map[string]interface{}{
			"id": "3", "type": "function_call_output", "call_id": "abc",
			"output": map[string]interface{}{"type": "text", "text": "ok"},
			"status": "completed",
		},
		map[string]interface{}{
			"role": "assistant", "id": "4",
			"content": []interface{}{
				map[string]interface{}{"type": "output_text", "text": "ok!"},
			},
		},
	}, false)

	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}
	if msgs[0]["role"] != "user" {
		t.Error("msg 0 should be user")
	}
	if msgs[1]["role"] != "assistant" {
		t.Error("msg 1 should be assistant")
	}
	if msgs[2]["role"] != "tool" {
		t.Error("msg 2 should be tool")
	}
	if msgs[3]["role"] != "assistant" {
		t.Error("msg 3 should be assistant")
	}
}

func TestTranslateInput_FunctionCallRCPropagates(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{
			"type": "function_call", "call_id": "c1", "name": "f",
			"arguments": "{}", "reasoning_content": "let me think",
		},
	}, true)

	if msgs[0]["reasoning_content"] != "let me think" {
		t.Errorf("expected 'let me think', got '%v'", msgs[0]["reasoning_content"])
	}
}

func TestTranslateInput_ReasoningRCPropagates(t *testing.T) {
	msgs, _ := translateInput([]interface{}{
		map[string]interface{}{"role": "assistant", "content": "I'll check"},
		map[string]interface{}{"type": "reasoning", "reasoning_content": "let me verify this"},
	}, true)

	if msgs[0]["reasoning_content"] != "let me verify this" {
		t.Errorf("expected 'let me verify this', got '%v'", msgs[0]["reasoning_content"])
	}
}

func TestTranslateInput_ObjectInput(t *testing.T) {
	msgs, _ := translateInput(map[string]interface{}{
		"role":    "user",
		"content": []interface{}{map[string]interface{}{"type": "input_text", "text": "hello object"}},
	}, false)

	// Object input is not directly handled by translateInput, but it shouldn't panic
	_ = msgs
}

// --- translateTools tests ---

func TestTranslateTools_Null(t *testing.T) {
	result := translateTools(nil)
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}

	result = translateTools([]interface{}{})
	if len(result) != 0 {
		t.Errorf("expected empty, got %v", result)
	}
}

func TestTranslateTools_Standard(t *testing.T) {
	result := translateTools([]interface{}{
		map[string]interface{}{
			"type": "function", "name": "s",
			"description": "d", "parameters": map[string]interface{}{},
		},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}
	if result[0]["function"].(map[string]interface{})["name"] != "s" {
		t.Error("tool name mismatch")
	}
}

func TestTranslateTools_FunctionWrapper(t *testing.T) {
	result := translateTools([]interface{}{
		map[string]interface{}{
			"function": map[string]interface{}{
				"name": "c", "description": "d",
			},
		},
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}
	if result[0]["function"].(map[string]interface{})["name"] != "c" {
		t.Error("tool name mismatch")
	}
}

func TestTranslateTools_FilterNameless(t *testing.T) {
	result := translateTools([]interface{}{
		map[string]interface{}{"type": "function"},
		map[string]interface{}{"type": "function", "name": "v"},
	})

	if len(result) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result))
	}
}

// --- translateToolChoice tests ---

func TestTranslateToolChoice_Null(t *testing.T) {
	if translateToolChoice(nil) != nil {
		t.Error("expected nil")
	}
}

func TestTranslateToolChoice_String(t *testing.T) {
	if translateToolChoice("auto") != "auto" {
		t.Error("expected 'auto'")
	}
}

func TestTranslateToolChoice_Object(t *testing.T) {
	result := translateToolChoice(map[string]interface{}{
		"type": "function", "name": "c",
	})

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("expected map result")
	}
	fn := resultMap["function"].(map[string]interface{})
	if fn["name"] != "c" {
		t.Error("function name mismatch")
	}
}

// --- lastUserText tests ---

func TestLastUserText_Found(t *testing.T) {
	result := lastUserText([]map[string]interface{}{
		{"role": "user", "content": "q1"},
		{"role": "assistant", "content": "a"},
		{"role": "user", "content": "q2"},
	})
	if result != "q2" {
		t.Errorf("expected 'q2', got '%s'", result)
	}
}

func TestLastUserText_Empty(t *testing.T) {
	if lastUserText([]map[string]interface{}{}) != "" {
		t.Error("expected empty string")
	}
}

func TestLastUserText_NoUser(t *testing.T) {
	result := lastUserText([]map[string]interface{}{
		{"role": "assistant", "content": "a"},
		{"role": "system", "content": "s"},
	})
	if result != "" {
		t.Errorf("expected '', got '%s'", result)
	}
}

// --- resolveModel tests ---

func TestResolveModel_Empty(t *testing.T) {
	if resolveModel("", "deepseek-v4-pro") != "deepseek-v4-pro" {
		t.Error("should return default")
	}
}

func TestResolveModel_GPT(t *testing.T) {
	if resolveModel("gpt-4o", "deepseek-v4-pro") != "deepseek-v4-pro" {
		t.Error("should override GPT models with default")
	}
}

func TestResolveModel_Other(t *testing.T) {
	if resolveModel("claude-4", "deepseek-v4-pro") != "claude-4" {
		t.Error("should pass through non-GPT models")
	}
}

// --- randomID tests ---

func TestRandomID_Format(t *testing.T) {
	id := randomID("test", 8)
	if len(id) < 10 {
		t.Errorf("expected long enough id, got %s (%d)", id, len(id))
	}
	if id[:4] != "test" {
		t.Errorf("expected prefix 'test', got %s", id[:4])
	}
}
