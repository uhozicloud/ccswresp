// Protocol Translation: OpenAI Responses API → Chat Completions API
// Handles message format conversion, tool call translation, and multi-modal content

import log from "./log.js";

/**
 * Extract plain text from various content formats.
 * Supports: string, content array, single content object.
 */
export function extractText(content) {
  if (typeof content === "string") return content;
  if (!content) return "";
  if (!Array.isArray(content)) {
    if (typeof content === "object" && content.type && content.text != null)
      return content.text;
    return "";
  }
  return content
    .filter((p) =>
      ["input_text", "output_text", "text", "reasoning_text"].includes(p.type)
    )
    .map((p) => p.text ?? "")
    .join("");
}

/**
 * Translate Responses API input array to Chat Completions messages array.
 *
 * @param {Array|string|null} input — Responses API "input" field
 * @param {Object} options
 * @param {boolean} options.keepReasoningContent — preserve reasoning_content in messages
 * @returns {{ messages: Array, stats: Object }}
 */
export function translateMessages(input, options = {}) {
  const messages = [];
  const stats = {
    skipped: { reasoning: 0, image: 0, file: 0, audio: 0, other: 0 },
    strippedReasoningContent: 0,
    preservedReasoningContent: 0,
  };

  // Handle non-array inputs
  if (!Array.isArray(input)) {
    if (typeof input === "string" && input.trim()) {
      messages.push({ role: "user", content: input });
    } else if (typeof input === "object" && input !== null) {
      const text = extractText(input.content);
      if (text) messages.push({ role: "user", content: text });
    }
    return { messages, stats };
  }

  for (const item of input) {
    if (!item) continue;

    // --- function_call → assistant tool_calls ---
    if (item.type === "function_call") {
      const last = messages[messages.length - 1];
      const target =
        last && last.role === "assistant"
          ? last
          : (() => {
              const m = { role: "assistant", tool_calls: [] };
              messages.push(m);
              return m;
            })();

      if (!target.tool_calls) target.tool_calls = [];
      target.tool_calls.push({
        id: item.call_id || item.id,
        type: "function",
        function: {
          name: item.name,
          arguments: item.arguments,
        },
      });

      if (item.status === "incomplete")
        log.warn("function_call status incomplete: " + (item.call_id || item.id));
      if (item.reasoning_content && !target.reasoning_content)
        target.reasoning_content = item.reasoning_content;
      continue;
    }

    // --- function_call_output → tool message ---
    if (item.type === "function_call_output") {
      messages.push({
        role: "tool",
        tool_call_id: item.call_id || item.id,
        content: extractText(item.output),
      });
      if (item.status === "incomplete")
        log.warn(
          "function_call_output status incomplete: " + (item.call_id || item.id)
        );
      continue;
    }

    // --- reasoning items ---
    if (item.type === "reasoning") {
      stats.skipped.reasoning++;
      if (item.reasoning_content) {
        const last = messages[messages.length - 1];
        if (last && !last.reasoning_content)
          last.reasoning_content = item.reasoning_content;
      }
      continue;
    }

    // --- role-based items (user, assistant, system, developer) ---
    if (item.role) {
      const role = item.role === "developer" ? "system" : item.role;
      const textContent = extractText(item.content);

      if (textContent) {
        const msg = { role, content: textContent };
        if (item.reasoning_content) msg.reasoning_content = item.reasoning_content;
        if (item.tool_calls) msg.tool_calls = item.tool_calls;
        if (item.tool_call_id) msg.tool_call_id = item.tool_call_id;
        messages.push(msg);
      }

      // Track skipped multi-modal parts
      if (Array.isArray(item.content)) {
        for (const p of item.content) {
          if (p.type === "input_image") stats.skipped.image++;
          if (p.type === "input_file") stats.skipped.file++;
          if (p.type === "input_audio") stats.skipped.audio++;
        }
      }
      continue;
    }

    // --- bare message type ---
    if (item.type === "message") {
      const textContent = extractText(item.content);
      if (textContent) messages.push({ role: "user", content: textContent });
      continue;
    }

    stats.skipped.other++;
  }

  // Handle reasoning_content preservation
  if (options.keepReasoningContent) {
    let count = 0;
    for (const msg of messages) {
      if (msg.reasoning_content) count++;
    }
    stats.preservedReasoningContent = count;
  } else {
    for (const msg of messages) {
      if (msg.reasoning_content) {
        delete msg.reasoning_content;
        stats.strippedReasoningContent++;
      }
    }
  }

  return { messages, stats };
}

/**
 * Get the text of the last user message in the conversation.
 */
export function lastUserText(messages) {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i]?.role === "user") return extractText(messages[i].content);
  }
  return "";
}

/**
 * Translate Responses API tools array to Chat Completions tools array.
 * Supports both { type, name, description, parameters } and
 * { function: { name, description, parameters } } formats.
 */
export function translateTools(rawTools) {
  if (!Array.isArray(rawTools)) return [];
  return rawTools
    .map((t) => {
      const name = t.name ?? t.function?.name;
      if (!name) return null;
      return {
        type: "function",
        function: {
          name,
          description: t.description ?? t.function?.description ?? "",
          parameters:
            t.parameters ??
            t.function?.parameters ?? { type: "object", properties: {} },
        },
      };
    })
    .filter(Boolean);
}

/**
 * Translate Responses API tool_choice to Chat Completions tool_choice.
 */
export function translateToolChoice(toolChoice) {
  if (!toolChoice) return null;
  if (typeof toolChoice === "string") return toolChoice;
  if (toolChoice.type === "function" && toolChoice.name)
    return { type: "function", function: { name: toolChoice.name } };
  return toolChoice;
}
