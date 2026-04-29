package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ============================================================
// Request: Claude Messages → OpenAI Chat Completions
// ============================================================

func convertClaudeToOpenAIRequest(cl map[string]any) map[string]any {
	oai := map[string]any{}
	oai["model"] = cl["model"]

	var msgs []map[string]any
	if sys := extractSystemText(cl["system"]); sys != "" {
		msgs = append(msgs, map[string]any{"role": "system", "content": sys})
	}
	if clMsgs, ok := cl["messages"].([]any); ok {
		for _, m := range clMsgs {
			if msg, ok := m.(map[string]any); ok {
				msgs = append(msgs, convertClaudeMsg(msg)...)
			}
		}
	}
	oai["messages"] = msgs

	if tools, ok := cl["tools"].([]any); ok && len(tools) > 0 {
		var oaiTools []any
		for _, t := range tools {
			if tool, ok := t.(map[string]any); ok {
				fn := map[string]any{"name": tool["name"]}
				if d, ok := tool["description"].(string); ok {
					fn["description"] = d
				}
				if s, ok := tool["input_schema"]; ok {
					fn["parameters"] = s
				}
				oaiTools = append(oaiTools, map[string]any{"type": "function", "function": fn})
			}
		}
		oai["tools"] = oaiTools
	}

	if tc, ok := cl["tool_choice"]; ok {
		oai["tool_choice"] = convertToolChoice(tc)
	}
	if v, ok := cl["max_tokens"]; ok {
		oai["max_completion_tokens"] = v
	}
	if v, ok := cl["stop_sequences"].([]any); ok {
		oai["stop"] = v
	}
	if v, ok := cl["temperature"]; ok {
		oai["temperature"] = v
	}
	if v, ok := cl["top_p"]; ok {
		oai["top_p"] = v
	}
	if stream, ok := cl["stream"].(bool); ok && stream {
		oai["stream"] = true
		oai["stream_options"] = map[string]any{"include_usage": true}
	}
	if meta, ok := cl["metadata"].(map[string]any); ok {
		if uid, ok := meta["user_id"].(string); ok {
			oai["user"] = uid
		}
	}
	return oai
}

func extractSystemText(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case []any:
		var parts []string
		for _, item := range s {
			if b, ok := item.(map[string]any); ok && b["type"] == "text" {
				if t, ok := b["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func convertClaudeMsg(msg map[string]any) []map[string]any {
	role, _ := msg["role"].(string)
	content := msg["content"]

	if s, ok := content.(string); ok {
		return []map[string]any{{"role": role, "content": s}}
	}
	blocks, ok := content.([]any)
	if !ok {
		return []map[string]any{{"role": role, "content": ""}}
	}
	if role == "assistant" {
		return convertAssistantBlocks(blocks)
	}
	return convertUserBlocks(blocks)
}

func convertAssistantBlocks(blocks []any) []map[string]any {
	var textParts []string
	var toolCalls []map[string]any
	for _, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			continue
		}
		switch block["type"] {
		case "text":
			if t, ok := block["text"].(string); ok {
				textParts = append(textParts, t)
			}
		case "tool_use":
			tc := map[string]any{
				"id":   block["id"],
				"type": "function",
				"function": map[string]any{
					"name":      block["name"],
					"arguments": jsonMarshal(block["input"]),
				},
			}
			toolCalls = append(toolCalls, tc)
		}
	}
	msg := map[string]any{"role": "assistant"}
	if len(textParts) > 0 {
		msg["content"] = strings.Join(textParts, "")
	} else if len(toolCalls) > 0 {
		msg["content"] = nil
	}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}
	return []map[string]any{msg}
}

func convertUserBlocks(blocks []any) []map[string]any {
	var result []map[string]any
	var textParts []string
	var imageParts []map[string]any

	for _, b := range blocks {
		block, ok := b.(map[string]any)
		if !ok {
			continue
		}
		switch block["type"] {
		case "text":
			if t, ok := block["text"].(string); ok {
				textParts = append(textParts, t)
			}
		case "image":
			if source, ok := block["source"].(map[string]any); ok && source["type"] == "base64" {
				mediaType, _ := source["media_type"].(string)
				data, _ := source["data"].(string)
				imageParts = append(imageParts, map[string]any{
					"type":      "image_url",
					"image_url": map[string]any{"url": fmt.Sprintf("data:%s;base64,%s", mediaType, data)},
				})
			}
		case "tool_result":
			toolUseID, _ := block["tool_use_id"].(string)
			result = append(result, map[string]any{
				"role":         "tool",
				"tool_call_id": toolUseID,
				"content":      extractTextContent(block["content"]),
			})
		}
	}

	if len(imageParts) > 0 {
		var parts []map[string]any
		for _, t := range textParts {
			parts = append(parts, map[string]any{"type": "text", "text": t})
		}
		parts = append(parts, imageParts...)
		result = append([]map[string]any{{"role": "user", "content": parts}}, result...)
	} else if len(textParts) > 0 {
		result = append([]map[string]any{{"role": "user", "content": strings.Join(textParts, "")}}, result...)
	}
	if len(result) == 0 {
		return []map[string]any{{"role": "user", "content": ""}}
	}
	return result
}

func extractTextContent(content any) string {
	switch c := content.(type) {
	case string:
		return c
	case []any:
		var parts []string
		for _, item := range c {
			if b, ok := item.(map[string]any); ok && b["type"] == "text" {
				if t, ok := b["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		return strings.Join(parts, "\n")
	}
	return ""
}

func convertToolChoice(tc any) any {
	switch v := tc.(type) {
	case string:
		return v
	case map[string]any:
		switch v["type"] {
		case "auto":
			return "auto"
		case "any":
			return "required"
		case "none":
			return "none"
		case "tool":
			name, _ := v["name"].(string)
			return map[string]any{"type": "function", "function": map[string]any{"name": name}}
		}
	}
	return "auto"
}

// ============================================================
// Response: OpenAI Chat → Claude Messages (non-streaming)
// ============================================================

func convertOpenAIResponseToClaude(oai map[string]any, model string) map[string]any {
	if errObj, ok := oai["error"].(map[string]any); ok {
		return map[string]any{
			"type":  "error",
			"error": map[string]any{"type": "api_error", "message": fmt.Sprintf("%v", errObj["message"])},
		}
	}

	choices, _ := oai["choices"].([]any)
	if len(choices) == 0 {
		return claudeError("no choices in response")
	}
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)

	var contentBlocks []map[string]any
	if text, ok := message["content"].(string); ok && text != "" {
		contentBlocks = append(contentBlocks, map[string]any{"type": "text", "text": text})
	}
	if toolCalls, ok := message["tool_calls"].([]any); ok {
		for _, tc := range toolCalls {
			if tcMap, ok := tc.(map[string]any); ok {
				fn, _ := tcMap["function"].(map[string]any)
				contentBlocks = append(contentBlocks, map[string]any{
					"type":  "tool_use",
					"id":    tcMap["id"],
					"name":  fn["name"],
					"input": parseJSONArgs(fn["arguments"]),
				})
			}
		}
	}
	if len(contentBlocks) == 0 {
		contentBlocks = append(contentBlocks, map[string]any{"type": "text", "text": ""})
	}

	stopReason := mapStopReason(safeString(choice["finish_reason"]))
	inputTokens, outputTokens := 0, 0
	if usage, ok := oai["usage"].(map[string]any); ok {
		inputTokens = toInt(usage["prompt_tokens"])
		outputTokens = toInt(usage["completion_tokens"])
	}

	return map[string]any{
		"id":            generateMsgID(),
		"type":          "message",
		"role":          "assistant",
		"content":       contentBlocks,
		"model":         model,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":                inputTokens,
			"output_tokens":               outputTokens,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
		},
	}
}

func mapStopReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	case "tool_calls":
		return "tool_use"
	default:
		return "end_turn"
	}
}

func claudeError(msg string) map[string]any {
	return map[string]any{
		"type":  "error",
		"error": map[string]any{"type": "api_error", "message": msg},
	}
}

// ============================================================
// Streaming SSE: OpenAI → Claude
// ============================================================

type StreamConverter struct {
	model        string
	msgID        string
	started      bool
	textBlockIdx int         // -1 = no text block open
	nextBlockIdx int         // next block index to assign
	toolBlocks   map[int]int // openai tool_call index → claude block index
	openTools    []int       // openai tool indices that have been started
	finishReason string
	outputTokens int
	inputTokens  int
}

func NewOpenAIToClaudeStreamConverter(model string) *StreamConverter {
	return &StreamConverter{
		model:        model,
		msgID:        generateMsgID(),
		textBlockIdx: -1,
		toolBlocks:   make(map[int]int),
	}
}

func (sc *StreamConverter) Convert(line string) []string {
	if strings.TrimSpace(line) == "" {
		return []string{""}
	}
	if !strings.HasPrefix(line, "data: ") {
		return []string{line}
	}
	data := strings.TrimPrefix(line, "data: ")
	if data == "[DONE]" {
		return sc.finalize()
	}

	var obj map[string]any
	if json.Unmarshal([]byte(data), &obj) != nil {
		return nil
	}

	// Extract usage from final chunks
	if usage, ok := obj["usage"].(map[string]any); ok {
		if v := toInt(usage["prompt_tokens"]); v > 0 {
			sc.inputTokens = v
		}
		if v := toInt(usage["completion_tokens"]); v > 0 {
			sc.outputTokens = v
		}
	}

	choices, _ := obj["choices"].([]any)
	if len(choices) == 0 {
		return nil
	}
	choice, _ := choices[0].(map[string]any)
	delta, _ := choice["delta"].(map[string]any)
	if delta == nil {
		return nil
	}

	var out []string
	if !sc.started {
		out = append(out, sc.emitMessageStart()...)
		sc.started = true
	}

	// Text content
	if content, ok := delta["content"].(string); ok && content != "" {
		if sc.textBlockIdx < 0 {
			sc.textBlockIdx = sc.nextBlockIdx
			sc.nextBlockIdx++
			out = append(out, sc.emitTextBlockStart(sc.textBlockIdx)...)
		}
		out = append(out, sc.emitTextDelta(sc.textBlockIdx, content)...)
	}

	// Tool calls
	if toolCalls, ok := delta["tool_calls"].([]any); ok {
		// Close text block if open
		if sc.textBlockIdx >= 0 {
			out = append(out, sc.emitBlockStop(sc.textBlockIdx))
			sc.textBlockIdx = -1
		}
		for _, tc := range toolCalls {
			tcMap, ok := tc.(map[string]any)
			if !ok {
				continue
			}
			tcIdx := toInt(tcMap["index"])
			if _, exists := sc.toolBlocks[tcIdx]; !exists {
				sc.toolBlocks[tcIdx] = sc.nextBlockIdx
				sc.nextBlockIdx++
				sc.openTools = append(sc.openTools, tcIdx)
				fn, _ := tcMap["function"].(map[string]any)
				name, _ := fn["name"].(string)
				toolID, _ := tcMap["id"].(string)
				out = append(out, sc.emitToolBlockStart(sc.toolBlocks[tcIdx], toolID, name)...)
			}
			fn, _ := tcMap["function"].(map[string]any)
			if args, ok := fn["arguments"].(string); ok && args != "" {
				out = append(out, sc.emitInputJsonDelta(sc.toolBlocks[tcIdx], args)...)
			}
		}
	}

	// Finish reason
	if fr, ok := choice["finish_reason"].(string); ok && fr != "" {
		sc.finishReason = fr
		if sc.textBlockIdx >= 0 {
			out = append(out, sc.emitBlockStop(sc.textBlockIdx))
			sc.textBlockIdx = -1
		}
		for _, idx := range sc.openTools {
			out = append(out, sc.emitBlockStop(sc.toolBlocks[idx]))
		}
		sc.openTools = nil
	}

	return out
}

func (sc *StreamConverter) finalize() []string {
	var out []string
	if sc.textBlockIdx >= 0 {
		out = append(out, sc.emitBlockStop(sc.textBlockIdx))
	}
	if !sc.started {
		out = append(out, sc.emitMessageStart()...)
	}
	for _, idx := range sc.openTools {
		out = append(out, sc.emitBlockStop(sc.toolBlocks[idx]))
	}

	stopReason := "end_turn"
	if sc.finishReason != "" {
		stopReason = mapStopReason(sc.finishReason)
	}
	out = append(out,
		"event: message_delta",
		fmt.Sprintf(`data: {"type":"message_delta","delta":{"stop_reason":"%s","stop_sequence":null},"usage":{"output_tokens":%d}}`, stopReason, sc.outputTokens),
		"",
		"event: message_stop",
		`data: {"type":"message_stop"}`,
		"",
	)
	return out
}

func (sc *StreamConverter) emitMessageStart() []string {
	return []string{
		"event: message_start",
		fmt.Sprintf(`data: {"type":"message_start","message":{"id":"%s","type":"message","role":"assistant","content":[],"model":"%s","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":%d,"output_tokens":1}}}`,
			sc.msgID, sc.model, sc.inputTokens),
		"",
	}
}

func (*StreamConverter) emitTextBlockStart(idx int) []string {
	return []string{
		"event: content_block_start",
		fmt.Sprintf(`data: {"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, idx),
		"",
	}
}

func (*StreamConverter) emitTextDelta(idx int, text string) []string {
	escaped, _ := json.Marshal(text)
	return []string{
		"event: content_block_delta",
		fmt.Sprintf(`data: {"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%s}}`, idx, string(escaped)),
		"",
	}
}

func (*StreamConverter) emitToolBlockStart(idx int, id, name string) []string {
	escapedName, _ := json.Marshal(name)
	return []string{
		"event: content_block_start",
		fmt.Sprintf(`data: {"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"%s","name":%s,"input":{}}}`, idx, id, string(escapedName)),
		"",
	}
}

func (*StreamConverter) emitInputJsonDelta(idx int, partial string) []string {
	escaped, _ := json.Marshal(partial)
	return []string{
		"event: content_block_delta",
		fmt.Sprintf(`data: {"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":%s}}`, idx, string(escaped)),
		"",
	}
}

func (*StreamConverter) emitBlockStop(idx int) string {
	return fmt.Sprintf("event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":%d}", idx)
}

// ============================================================
// Helpers
// ============================================================

func generateMsgID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return "msg_" + hex.EncodeToString(b)
}

func jsonMarshal(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func parseJSONArgs(v any) map[string]any {
	s, ok := v.(string)
	if !ok {
		return map[string]any{}
	}
	var result map[string]any
	if json.Unmarshal([]byte(s), &result) != nil {
		return map[string]any{}
	}
	return result
}

func toInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

func safeString(v any) string {
	s, _ := v.(string)
	return s
}

// ============================================================
// Response: Claude Messages → OpenAI Chat Completions (non-streaming)
// ============================================================

func convertClaudeResponseToOpenAI(cl map[string]any, model string) map[string]any {
	if errObj, ok := cl["error"].(map[string]any); ok {
		return map[string]any{
			"error": map[string]any{"message": fmt.Sprintf("%v", errObj["message"]), "type": "api_error"},
		}
	}

	content, _ := cl["content"].([]any)
	var text string
	var toolCalls []map[string]any
	for _, block := range content {
		b, ok := block.(map[string]any)
		if !ok {
			continue
		}
		switch b["type"] {
		case "text":
			if t, ok := b["text"].(string); ok {
				text += t
			}
		case "tool_use":
			tc := map[string]any{
				"id":   b["id"],
				"type": "function",
				"function": map[string]any{
					"name":      b["name"],
					"arguments": jsonMarshal(b["input"]),
				},
			}
			toolCalls = append(toolCalls, tc)
		}
	}

	msg := map[string]any{"role": "assistant", "content": text}
	if len(toolCalls) > 0 {
		msg["tool_calls"] = toolCalls
	}

	stopReason := mapClaudeStopReason(safeString(cl["stop_reason"]))
	inputTokens, outputTokens := 0, 0
	if usage, ok := cl["usage"].(map[string]any); ok {
		inputTokens = toInt(usage["input_tokens"])
		outputTokens = toInt(usage["output_tokens"])
	}

	return map[string]any{
		"id":      "chatcmpl-" + generateMsgID(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       msg,
			"finish_reason": stopReason,
		}},
		"usage": map[string]any{
			"prompt_tokens":     inputTokens,
			"completion_tokens": outputTokens,
		},
	}
}

func mapClaudeStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "tool_use":
		return "tool_calls"
	default:
		return "stop"
	}
}

// ============================================================
// Streaming SSE: Claude → OpenAI
// ============================================================

type ClaudeToOpenAIStreamConverter struct {
	model        string
	chunkID      string
	started      bool
	inputTokens  int
	outputTokens int
}

func NewClaudeToOpenAIStreamConverter(model string) *ClaudeToOpenAIStreamConverter {
	return &ClaudeToOpenAIStreamConverter{
		model:   model,
		chunkID: "chatcmpl-" + generateMsgID(),
	}
}

func (sc *ClaudeToOpenAIStreamConverter) Convert(line string) []string {
	if strings.TrimSpace(line) == "" {
		return []string{""}
	}
	if !strings.HasPrefix(line, "data: ") && !strings.HasPrefix(line, "event: ") {
		return nil
	}

	if strings.HasPrefix(line, "event: ") {
		return nil
	}

	data := strings.TrimPrefix(line, "data: ")
	var obj map[string]any
	if json.Unmarshal([]byte(data), &obj) != nil {
		return nil
	}

	msgType, _ := obj["type"].(string)

	var out []string

	switch msgType {
	case "message_start":
		if msg, ok := obj["message"].(map[string]any); ok {
			if usage, ok := msg["usage"].(map[string]any); ok {
				sc.inputTokens = toInt(usage["input_tokens"])
			}
		}
		out = append(out, sc.emitChunk(map[string]any{"role": "assistant"}, nil))
		sc.started = true

	case "content_block_start":
		block, _ := obj["content_block"].(map[string]any)
		if block == nil {
			return nil
		}
		switch block["type"] {
		case "tool_use":
			id, _ := block["id"].(string)
			name, _ := block["name"].(string)
			tc := map[string]any{
				"index": toInt(obj["index"]),
				"id":    id,
				"type":  "function",
				"function": map[string]any{
					"name":      name,
					"arguments": "",
				},
			}
			out = append(out, sc.emitChunk(nil, []map[string]any{tc}))
		}

	case "content_block_delta":
		delta, _ := obj["delta"].(map[string]any)
		if delta == nil {
			return nil
		}
		switch delta["type"] {
		case "text_delta":
			text, _ := delta["text"].(string)
			out = append(out, sc.emitChunk(map[string]any{"content": text}, nil))
		case "input_json_delta":
			partial, _ := delta["partial_json"].(string)
			idx := toInt(obj["index"])
			tc := map[string]any{
				"index": idx,
				"function": map[string]any{
					"arguments": partial,
				},
			}
			out = append(out, sc.emitChunk(nil, []map[string]any{tc}))
		}

	case "message_delta":
		delta, _ := obj["delta"].(map[string]any)
		if usage, ok := obj["usage"].(map[string]any); ok {
			sc.outputTokens = toInt(usage["output_tokens"])
		}
		finishReason := "stop"
		if delta != nil {
			finishReason = mapClaudeStopReason(safeString(delta["stop_reason"]))
		}
		out = append(out, sc.emitFinishChunk(finishReason))

	case "message_stop":
		out = append(out, "data: [DONE]", "")
	}

	return out
}

func (sc *ClaudeToOpenAIStreamConverter) emitChunk(delta map[string]any, toolCalls []map[string]any) string {
	d := map[string]any{}
	if delta != nil {
		for k, v := range delta {
			d[k] = v
		}
	}
	if len(toolCalls) > 0 {
		d["tool_calls"] = toolCalls
	}
	chunk := map[string]any{
		"id":      sc.chunkID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   sc.model,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         d,
			"finish_reason": nil,
		}},
	}
	b, _ := json.Marshal(chunk)
	return "data: " + string(b)
}

func (sc *ClaudeToOpenAIStreamConverter) emitFinishChunk(finishReason string) string {
	chunk := map[string]any{
		"id":      sc.chunkID,
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   sc.model,
		"choices": []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": finishReason,
		}},
		"usage": map[string]any{
			"prompt_tokens":     sc.inputTokens,
			"completion_tokens": sc.outputTokens,
		},
	}
	b, _ := json.Marshal(chunk)
	return "data: " + string(b)
}
