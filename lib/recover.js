// Reasoning Content Recovery
// Manages reasoning_content across multi-turn tool-call conversations.
// When a Chat Completions API strips reasoning_content from tool-call
// assistant messages, we remember the last response's reasoning and
// restore it into subsequent requests so thinking mode stays active.

const reasoningQueue = [];

/**
 * Remember reasoning_content from a completed response.
 * Stores it in a FIFO queue for later recovery.
 *
 * @param {string} key - session identifier (unused currently; reserved for multi-session support)
 * @param {Array} messages - assistant messages with reasoning_content
 */
export function rememberReasoning(key, messages) {
  for (const msg of messages) {
    if (msg.role === "assistant" && msg.reasoning_content) {
      reasoningQueue.push(msg.reasoning_content);
    }
  }
}

/**
 * Recover reasoning_content into messages that are missing it.
 * Tool-call assistant messages often lose their reasoning_content
 * when passed back by the Responses API. We restore it from the queue.
 *
 * @param {string} key - session identifier
 * @param {Array} messages - the messages array to restore into (mutated in place)
 * @returns {number} count of restored messages
 */
export function recoverReasoning(key, messages) {
  if (reasoningQueue.length === 0) return 0;

  let recovered = 0;
  for (const msg of messages) {
    if (
      msg.role === "assistant" &&
      msg.tool_calls &&
      !msg.reasoning_content
    ) {
      msg.reasoning_content =
        reasoningQueue[Math.min(recovered, reasoningQueue.length - 1)];
      recovered++;
    }
  }
  return recovered;
}

/**
 * Derive a session key from the request body.
 * Currently returns a global key; reserved for future per-session isolation.
 */
export function sessionKey(body) {
  return "g";
}
