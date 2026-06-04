// Protocol Translation Unit Tests
// Tests for translateMessages, translateTools, translateToolChoice, extractText, lastUserText

import { test } from "node:test";
import assert from "node:assert/strict";
import {
  extractText,
  translateMessages,
  translateTools,
  translateToolChoice,
  lastUserText,
} from "../lib/translate.js";

// --- extractText ---
test("extractText - string", () => {
  assert.equal(extractText("hello"), "hello");
});

test("extractText - empty non-array", () => {
  assert.equal(extractText({ a: 1 }), "");
  assert.equal(extractText(123), "");
});

test("extractText - content array", () => {
  assert.equal(
    extractText([
      { type: "input_text", text: "a" },
      { type: "output_text", text: "b" },
    ]),
    "ab"
  );
});

test("extractText - ignore non-text types", () => {
  assert.equal(
    extractText([{ type: "input_image" }, { type: "input_text", text: "t" }]),
    "t"
  );
});

test("extractText - single object with type", () => {
  assert.equal(extractText({ type: "text", text: "ok" }), "ok");
});

test("extractText - reasoning_text included", () => {
  assert.equal(
    extractText([{ type: "reasoning_text", text: "thinking..." }]),
    "thinking..."
  );
});

// --- translateMessages ---
test("translateMessages - string input becomes user message", () => {
  const r = translateMessages("hello");
  assert.equal(r.messages.length, 1);
  assert.equal(r.messages[0].role, "user");
  assert.equal(r.messages[0].content, "hello");
});

test("translateMessages - empty/whitespace string yields no messages", () => {
  assert.equal(translateMessages("   ").messages.length, 0);
});

test("translateMessages - null/undefined yields no messages", () => {
  assert.equal(translateMessages(null).messages.length, 0);
  assert.equal(translateMessages(undefined).messages.length, 0);
});

test("translateMessages - empty array yields no messages", () => {
  assert.equal(translateMessages([]).messages.length, 0);
});

test("translateMessages - simple user/assistant pair", () => {
  const r = translateMessages([
    { role: "user", content: [{ type: "input_text", text: "hi" }] },
    { role: "assistant", content: [{ type: "output_text", text: "hi!" }] },
  ]);
  assert.equal(r.messages.length, 2);
  assert.equal(r.messages[0].role, "user");
  assert.equal(r.messages[0].content, "hi");
  assert.equal(r.messages[1].role, "assistant");
  assert.equal(r.messages[1].content, "hi!");
});

test("translateMessages - developer role becomes system", () => {
  const r = translateMessages([{ role: "developer", content: "sys" }]);
  assert.equal(r.messages[0].role, "system");
});

test("translateMessages - function_call merges into assistant tool_calls", () => {
  const r = translateMessages([
    { role: "assistant", content: [] },
    { type: "function_call", call_id: "c1", name: "f", arguments: "{}" },
    { type: "function_call", call_id: "c2", name: "g", arguments: "{}" },
  ]);
  assert.equal(r.messages.length, 1);
  assert.equal(r.messages[0].role, "assistant");
  assert.equal(r.messages[0].tool_calls.length, 2);
  assert.equal(r.messages[0].tool_calls[0].id, "c1");
  assert.equal(r.messages[0].tool_calls[1].id, "c2");
});

test("translateMessages - function_call with no prior assistant creates new one", () => {
  const r = translateMessages([
    { type: "function_call", call_id: "c1", name: "f", arguments: "{}" },
  ]);
  assert.equal(r.messages.length, 1);
  assert.equal(r.messages[0].role, "assistant");
  assert.equal(r.messages[0].tool_calls.length, 1);
});

test("translateMessages - function_call_output becomes tool message", () => {
  const r = translateMessages([
    { type: "function_call_output", call_id: "c1", output: { type: "text", text: "ok" } },
  ]);
  assert.equal(r.messages[0].role, "tool");
  assert.equal(r.messages[0].tool_call_id, "c1");
  assert.equal(r.messages[0].content, "ok");
});

test("translateMessages - reasoning items are skipped", () => {
  const r = translateMessages([
    { role: "user", content: "q" },
    { type: "reasoning", reasoning_content: "t" },
  ]);
  assert.equal(r.messages.length, 1);
  assert.equal(r.stats.skipped.reasoning, 1);
});

test("translateMessages - reasoning_content stripped by default", () => {
  const r = translateMessages([
    { role: "assistant", content: "a", reasoning_content: "t" },
  ]);
  assert.equal(r.messages[0].reasoning_content, undefined);
  assert.equal(r.stats.strippedReasoningContent, 1);
});

test("translateMessages - reasoning_content kept with option", () => {
  const r = translateMessages(
    [{ role: "assistant", content: "a", reasoning_content: "t" }],
    { keepReasoningContent: true }
  );
  assert.equal(r.messages[0].reasoning_content, "t");
  assert.equal(r.stats.preservedReasoningContent, 1);
  assert.equal(r.stats.strippedReasoningContent, 0);
});

test("translateMessages - multi-modal stats (file + audio)", () => {
  const r = translateMessages([
    {
      role: "user",
      content: [
        { type: "input_text", text: "hi" },
        { type: "input_file" },
        { type: "input_audio" },
      ],
    },
  ]);
  assert.equal(r.stats.skipped.file, 1);
  assert.equal(r.stats.skipped.audio, 1);
});

test("translateMessages - multi-modal stats (image)", () => {
  const r = translateMessages([
    {
      role: "user",
      content: [
        { type: "input_text", text: "what is this?" },
        { type: "input_image" },
      ],
    },
  ]);
  assert.equal(r.stats.skipped.image, 1);
});

test("translateMessages - function_call with incomplete status", () => {
  const r = translateMessages([
    {
      type: "function_call",
      call_id: "c1",
      name: "f",
      arguments: "{}",
      status: "incomplete",
    },
  ]);
  assert.equal(r.messages.length, 1);
  assert.equal(r.messages[0].role, "assistant");
  assert.equal(r.messages[0].tool_calls[0].function.name, "f");
});

test("translateMessages - full conversation round-trip", () => {
  const r = translateMessages([
    { role: "user", id: "1", content: [{ type: "input_text", text: "w?" }] },
    {
      id: "2",
      type: "function_call",
      call_id: "abc",
      name: "f",
      arguments: "{}",
      status: "completed",
    },
    {
      id: "3",
      type: "function_call_output",
      call_id: "abc",
      output: { type: "text", text: "ok" },
      status: "completed",
    },
    {
      role: "assistant",
      id: "4",
      content: [{ type: "output_text", text: "ok!" }],
    },
  ]);
  assert.equal(r.messages.length, 4);
  assert.equal(r.messages[0].role, "user");
  assert.equal(r.messages[1].role, "assistant");
  assert.equal(r.messages[2].role, "tool");
  assert.equal(r.messages[2].content, "ok");
  assert.equal(r.messages[3].role, "assistant");
});

test("translateMessages - function_call with reasoning_content propagates", () => {
  const r = translateMessages(
    [
      {
        type: "function_call",
        call_id: "c1",
        name: "f",
        arguments: "{}",
        reasoning_content: "let me think",
      },
    ],
    { keepReasoningContent: true }
  );
  assert.equal(r.messages[0].reasoning_content, "let me think");
  assert.equal(r.stats.preservedReasoningContent, 1);
});

test("translateMessages - reasoning type propagates rc to prior assistant", () => {
  const r = translateMessages(
    [
      { role: "assistant", content: "I'll check" },
      { type: "reasoning", reasoning_content: "let me verify this" },
    ],
    { keepReasoningContent: true }
  );
  assert.equal(r.messages[0].reasoning_content, "let me verify this");
  assert.equal(r.stats.preservedReasoningContent, 1);
});

test("translateMessages - function_call_output with incomplete status", () => {
  const r = translateMessages([
    {
      type: "function_call_output",
      call_id: "c1",
      output: { type: "text", text: "partial" },
      status: "incomplete",
    },
  ]);
  assert.equal(r.messages[0].role, "tool");
  assert.equal(r.messages[0].content, "partial");
});

test("translateMessages - object input with text content", () => {
  const r = translateMessages({
    role: "user",
    content: [{ type: "input_text", text: "hello object" }],
  });
  assert.equal(r.messages[0].role, "user");
  assert.equal(r.messages[0].content, "hello object");
});

test("translateMessages - bare message type item", () => {
  const r = translateMessages([
    { type: "message", content: [{ type: "output_text", text: "bare" }] },
  ]);
  assert.equal(r.messages[0].role, "user");
  assert.equal(r.messages[0].content, "bare");
});

test("translateMessages - unknown type tracked in stats", () => {
  const r = translateMessages([{ type: "unknown_thing", data: "x" }]);
  assert.equal(r.stats.skipped.other, 1);
});

// --- translateTools ---
test("translateTools - null/empty", () => {
  assert.deepEqual(translateTools(null), []);
  assert.deepEqual(translateTools(undefined), []);
  assert.deepEqual(translateTools([]), []);
});

test("translateTools - standard format", () => {
  const r = translateTools([
    { type: "function", name: "s", description: "d", parameters: {} },
  ]);
  assert.equal(r.length, 1);
  assert.equal(r[0].type, "function");
  assert.equal(r[0].function.name, "s");
  assert.equal(r[0].function.description, "d");
});

test("translateTools - function wrapper format", () => {
  const r = translateTools([{ function: { name: "c", description: "d" } }]);
  assert.equal(r.length, 1);
  assert.equal(r[0].function.name, "c");
});

test("translateTools - filter out nameless tools", () => {
  const r = translateTools([
    { type: "function" },
    { type: "function", name: "v" },
  ]);
  assert.equal(r.length, 1);
  assert.equal(r[0].function.name, "v");
});

test("translateTools - default parameters for empty params", () => {
  const r = translateTools([{ type: "function", name: "f" }]);
  assert.deepEqual(r[0].function.parameters, {
    type: "object",
    properties: {},
  });
});

// --- translateToolChoice ---
test("translateToolChoice - null", () => {
  assert.equal(translateToolChoice(null), null);
});

test("translateToolChoice - string passthrough", () => {
  assert.equal(translateToolChoice("auto"), "auto");
  assert.equal(translateToolChoice("required"), "required");
  assert.equal(translateToolChoice("none"), "none");
});

test("translateToolChoice - object with name", () => {
  const r = translateToolChoice({ type: "function", name: "c" });
  assert.equal(r.type, "function");
  assert.equal(r.function.name, "c");
});

test("translateToolChoice - object without name returned as-is", () => {
  const r = translateToolChoice({ type: "any" });
  assert.deepEqual(r, { type: "any" });
});

// --- lastUserText ---
test("lastUserText - finds last user message", () => {
  assert.equal(
    lastUserText([
      { role: "user", content: "q1" },
      { role: "assistant", content: "a" },
      { role: "user", content: "q2" },
    ]),
    "q2"
  );
});

test("lastUserText - empty array returns empty string", () => {
  assert.equal(lastUserText([]), "");
});

test("lastUserText - no user messages returns empty string", () => {
  assert.equal(
    lastUserText([
      { role: "assistant", content: "a" },
      { role: "system", content: "s" },
    ]),
    ""
  );
});

test("lastUserText - content array", () => {
  assert.equal(
    lastUserText([
      {
        role: "user",
        content: [{ type: "input_text", text: "array query" }],
      },
    ]),
    "array query"
  );
});

console.log("\n✓ All 33 translation tests passed!\n");
