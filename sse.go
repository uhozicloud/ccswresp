package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// sseTranslator handles streaming SSE event translation:
// Chat Completions SSE chunks → Responses API SSE events.
type sseTranslator struct {
	w               http.ResponseWriter
	model           string
	responseID      string
	messageItemID   string
	textStarted     bool
	contentSoFar    string
	reasoningStarted bool
	reasoningSoFar  string
	reasoningItemID string
	toolCalls       map[int]*toolCall
	started         bool
	outputItemCount int
	outputItems     []outputItem
	lastUsage       *usageInfo
}

type toolCall struct {
	id        string
	name      string
	arguments string
}

type outputItem struct {
	index  int
	typ    string
	itemID string
}

type usageInfo struct {
	promptTokens     int
	completionTokens int
	totalTokens      int
}

func newSseTranslator(w http.ResponseWriter, model string) *sseTranslator {
	return &sseTranslator{
		w:              w,
		model:          model,
		responseID:     randomID("resp", 8),
		messageItemID:  randomID("item", 8),
		toolCalls:      make(map[int]*toolCall),
	}
}

func (t *sseTranslator) emit(event string, data interface{}) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(t.w, "event: %s\ndata: %s\n\n", event, string(jsonBytes))
	if f, ok := t.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (t *sseTranslator) ensureStarted() {
	if t.started {
		return
	}
	t.started = true

	t.emit("response.created", map[string]interface{}{
		"type": "response.created",
		"response": map[string]interface{}{
			"id":     t.responseID,
			"object": "response",
			"status": "in_progress",
			"model":  t.model,
			"output": []interface{}{},
		},
	})
	t.emit("response.in_progress", map[string]interface{}{
		"type":        "response.in_progress",
		"response_id": t.responseID,
	})
	logInfo("SSE start: " + t.responseID)
}

// feed processes a single Chat Completions SSE chunk.
func (t *sseTranslator) feed(chunkMap map[string]interface{}) {
	choices, ok := chunkMap["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return
	}

	delta, ok := choice["delta"].(map[string]interface{})
	if !ok {
		return
	}

	// Track usage
	if usageRaw, ok := chunkMap["usage"].(map[string]interface{}); ok {
		u := &usageInfo{}
		if v, ok := usageRaw["prompt_tokens"].(float64); ok {
			u.promptTokens = int(v)
		}
		if v, ok := usageRaw["completion_tokens"].(float64); ok {
			u.completionTokens = int(v)
		}
		if v, ok := usageRaw["total_tokens"].(float64); ok {
			u.totalTokens = int(v)
		}
		t.lastUsage = u
	}

	// Text content delta
	if content, ok := delta["content"].(string); ok && content != "" {
		t.ensureStarted()
		t.contentSoFar += content

		if !t.textStarted {
			t.textStarted = true
			oi := t.outputItemCount
			t.outputItemCount++
			t.outputItems = append(t.outputItems, outputItem{index: oi, typ: "message", itemID: t.messageItemID})

			t.emit("response.output_item.added", map[string]interface{}{
				"type":         "response.output_item.added",
				"response_id":  t.responseID,
				"output_index": oi,
				"item": map[string]interface{}{
					"id":     t.messageItemID,
					"type":   "message",
					"role":   "assistant",
					"status": "in_progress",
					"content": []interface{}{},
				},
			})
			t.emit("response.content_part.added", map[string]interface{}{
				"type":          "response.content_part.added",
				"response_id":   t.responseID,
				"item_id":       t.messageItemID,
				"output_index":  oi,
				"content_index": 0,
				"part": map[string]interface{}{
					"type":        "output_text",
					"text":        "",
					"annotations": []interface{}{},
				},
			})
		}

		t.emit("response.output_text.delta", map[string]interface{}{
			"type":          "response.output_text.delta",
			"response_id":   t.responseID,
			"item_id":       t.messageItemID,
			"output_index":  t.msgIndex(),
			"content_index": 0,
			"delta":         content,
		})
	}

	// Reasoning content delta
	if rc, ok := delta["reasoning_content"].(string); ok && rc != "" {
		t.ensureStarted()
		t.reasoningSoFar += rc

		if !t.reasoningStarted {
			t.reasoningStarted = true
			t.reasoningItemID = randomID("rsn", 8)
			oi := t.outputItemCount
			t.outputItemCount++
			t.outputItems = append(t.outputItems, outputItem{index: oi, typ: "reasoning", itemID: t.reasoningItemID})

			t.emit("response.output_item.added", map[string]interface{}{
				"type":         "response.output_item.added",
				"response_id":  t.responseID,
				"output_index": oi,
				"item": map[string]interface{}{
					"id":      t.reasoningItemID,
					"type":    "reasoning",
					"status":  "in_progress",
					"summary": []interface{}{},
				},
			})
			t.emit("response.content_part.added", map[string]interface{}{
				"type":          "response.content_part.added",
				"response_id":   t.responseID,
				"item_id":       t.reasoningItemID,
				"output_index":  oi,
				"content_index": 0,
				"part": map[string]interface{}{
					"type": "reasoning_text",
					"text": "",
				},
			})
		}

		rIdx := t.rsnIndex()
		if rIdx >= 0 {
			t.emit("response.reasoning_text.delta", map[string]interface{}{
				"type":          "response.reasoning_text.delta",
				"response_id":   t.responseID,
				"item_id":       t.reasoningItemID,
				"output_index":  rIdx,
				"content_index": 0,
				"delta":         rc,
			})
		}
	}

	// Tool call deltas
	if rawTCs, ok := delta["tool_calls"].([]interface{}); ok {
		t.ensureStarted()
		for _, rawTC := range rawTCs {
			tcMap, ok := rawTC.(map[string]interface{})
			if !ok {
				continue
			}

			index := 0
			if idx, ok := tcMap["index"].(float64); ok {
				index = int(idx)
			}

			if _, exists := t.toolCalls[index]; !exists {
				id := ""
				if idStr, ok := tcMap["id"].(string); ok {
					id = idStr
				} else {
					id = fmt.Sprintf("call_%d", index)
				}

				name := ""
				if fn, ok := tcMap["function"].(map[string]interface{}); ok {
					if n, ok := fn["name"].(string); ok {
						name = n
					}
				}

				call := &toolCall{id: id, name: name}
				t.toolCalls[index] = call

				oi := t.outputItemCount
				t.outputItemCount++
				t.outputItems = append(t.outputItems, outputItem{index: oi, typ: "function_call", itemID: "fc_" + id})

				t.emit("response.output_item.added", map[string]interface{}{
					"type":         "response.output_item.added",
					"response_id":  t.responseID,
					"output_index": oi,
					"item": map[string]interface{}{
						"id":      "fc_" + id,
						"type":    "function_call",
						"call_id": id,
						"name":    name,
						"status":  "in_progress",
					},
				})
				logInfo("tool: " + name + " (" + id + ")")
			}

			call := t.toolCalls[index]

			if fn, ok := tcMap["function"].(map[string]interface{}); ok {
				if n, ok := fn["name"].(string); ok && n != "" {
					call.name = n
				}
				if args, ok := fn["arguments"].(string); ok && args != "" {
					call.arguments += args
					oi := t.itemIndex("fc_" + call.id)
					if oi >= 0 {
						t.emit("response.function_call_arguments.delta", map[string]interface{}{
							"type":          "response.function_call_arguments.delta",
							"response_id":   t.responseID,
							"item_id":       "fc_" + call.id,
							"output_index":  oi,
							"delta":         args,
						})
					}
				}
			}
		}
	}
}

// done completes the SSE stream, emitting all done/completed events.
func (t *sseTranslator) done(usageOverride *usageInfo) {
	t.ensureStarted()
	usage := t.lastUsage
	if usageOverride != nil {
		usage = usageOverride
	}

	// Close text output
	if t.textStarted {
		oi := t.msgIndex()
		t.emit("response.content_part.done", map[string]interface{}{
			"type":          "response.content_part.done",
			"response_id":   t.responseID,
			"item_id":       t.messageItemID,
			"output_index":  oi,
			"content_index": 0,
			"part": map[string]interface{}{
				"type":        "output_text",
				"text":        t.contentSoFar,
				"annotations": []interface{}{},
			},
		})
		t.emit("response.output_text.done", map[string]interface{}{
			"type":          "response.output_text.done",
			"response_id":   t.responseID,
			"item_id":       t.messageItemID,
			"output_index":  oi,
			"content_index": 0,
			"text":          t.contentSoFar,
		})
		t.emit("response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"response_id":  t.responseID,
			"output_index": oi,
			"item": map[string]interface{}{
				"id":     t.messageItemID,
				"type":   "message",
				"role":   "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "output_text",
						"text":        t.contentSoFar,
						"annotations": []interface{}{},
					},
				},
				"status": "completed",
			},
		})
		logResp(fmt.Sprintf("text output: %d chars", len(t.contentSoFar)))
	}

	// Close reasoning output
	if t.reasoningStarted {
		oi := t.rsnIndex()
		t.emit("response.content_part.done", map[string]interface{}{
			"type":          "response.content_part.done",
			"response_id":   t.responseID,
			"item_id":       t.reasoningItemID,
			"output_index":  oi,
			"content_index": 0,
			"part": map[string]interface{}{
				"type": "reasoning_text",
				"text": t.reasoningSoFar,
			},
		})
		t.emit("response.reasoning_text.done", map[string]interface{}{
			"type":          "response.reasoning_text.done",
			"response_id":   t.responseID,
			"item_id":       t.reasoningItemID,
			"output_index":  oi,
			"content_index": 0,
			"text":          t.reasoningSoFar,
		})
		t.emit("response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"response_id":  t.responseID,
			"output_index": oi,
			"item": map[string]interface{}{
				"id":   t.reasoningItemID,
				"type": "reasoning",
				"content": []interface{}{
					map[string]interface{}{
						"type": "reasoning_text",
						"text": t.reasoningSoFar,
					},
				},
				"status": "completed",
			},
		})
		logResp(fmt.Sprintf("reasoning output: %d chars", len(t.reasoningSoFar)))
	}

	// Close tool calls
	for idx, call := range t.toolCalls {
		oi := t.itemIndex("fc_" + call.id)
		outIdx := oi
		if outIdx < 0 {
			outIdx = idx + 1
		}
		t.emit("response.function_call_arguments.done", map[string]interface{}{
			"type":          "response.function_call_arguments.done",
			"response_id":   t.responseID,
			"item_id":       "fc_" + call.id,
			"output_index":  outIdx,
			"arguments":     call.arguments,
			"name":          call.name,
			"call_id":       call.id,
		})
		t.emit("response.output_item.done", map[string]interface{}{
			"type":         "response.output_item.done",
			"response_id":  t.responseID,
			"output_index": outIdx,
			"item": map[string]interface{}{
				"id":        "fc_" + call.id,
				"type":      "function_call",
				"call_id":   call.id,
				"name":      call.name,
				"arguments": call.arguments,
				"status":    "completed",
			},
		})
		logResp("tool done: " + call.name)
	}

	// Build output snapshot
	outSnapshot := t.buildOutputSnapshot()

	var respUsage interface{}
	if usage != nil {
		respUsage = map[string]interface{}{
			"input_tokens":  usage.promptTokens,
			"output_tokens": usage.completionTokens,
			"total_tokens":  usage.totalTokens,
		}
	}

	t.emit("response.completed", map[string]interface{}{
		"type": "response.completed",
		"response": map[string]interface{}{
			"id":     t.responseID,
			"object": "response",
			"status": "completed",
			"model":  t.model,
			"output": outSnapshot,
			"usage":  respUsage,
		},
	})

	if usage != nil {
		logToks(usage.promptTokens, usage.completionTokens, usage.totalTokens)
	}
	logOk("SSE done: " + t.responseID)
}

func (t *sseTranslator) buildOutputSnapshot() []interface{} {
	var snapshot []interface{}
	for _, o := range t.outputItems {
		switch o.typ {
		case "message":
			snapshot = append(snapshot, map[string]interface{}{
				"id":   o.itemID,
				"type": "message",
				"role": "assistant",
				"content": []interface{}{
					map[string]interface{}{
						"type":        "output_text",
						"text":        t.contentSoFar,
						"annotations": []interface{}{},
					},
				},
				"status": "completed",
			})
		case "reasoning":
			snapshot = append(snapshot, map[string]interface{}{
				"id":   o.itemID,
				"type": "reasoning",
				"content": []interface{}{
					map[string]interface{}{
						"type": "reasoning_text",
						"text": t.reasoningSoFar,
					},
				},
				"status": "completed",
			})
		case "function_call":
			for _, call := range t.toolCalls {
				if "fc_"+call.id == o.itemID {
					snapshot = append(snapshot, map[string]interface{}{
						"id":        o.itemID,
						"type":      "function_call",
						"call_id":   call.id,
						"name":      call.name,
						"arguments": call.arguments,
						"status":    "completed",
					})
				}
			}
		}
	}
	return snapshot
}

func (t *sseTranslator) error(msg string) {
	t.emit("error", map[string]interface{}{
		"type":    "error",
		"code":    "proxy_error",
		"message": msg,
	})
	logErr("SSE error: " + msg)
}

func (t *sseTranslator) msgIndex() int {
	for _, o := range t.outputItems {
		if o.typ == "message" {
			return o.index
		}
	}
	return 0
}

func (t *sseTranslator) rsnIndex() int {
	for _, o := range t.outputItems {
		if o.typ == "reasoning" {
			return o.index
		}
	}
	return -1
}

func (t *sseTranslator) itemIndex(id string) int {
	for _, o := range t.outputItems {
		if o.itemID == id {
			return o.index
		}
	}
	return -1
}
