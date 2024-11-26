// extract the errors in the package to the top level:

package anthropicclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

var (
	ErrInvalidEventType           = fmt.Errorf("invalid event type field type")
	ErrInvalidMessageField        = fmt.Errorf("invalid message field type")
	ErrInvalidUsageField          = fmt.Errorf("invalid usage field type")
	ErrInvalidIndexField          = fmt.Errorf("invalid index field type")
	ErrInvalidDeltaField          = fmt.Errorf("invalid delta field type")
	ErrInvalidDeltaTypeField      = fmt.Errorf("invalid delta type field type")
	ErrInvalidDeltaTextField      = fmt.Errorf("invalid delta text field type")
	ErrInvalidDeltaInputJSONField = fmt.Errorf("invalid delta input JSON field type")
	ErrContentIndexOutOfRange     = fmt.Errorf("content index out of range")
	ErrFailedCastToTextContent    = fmt.Errorf("failed to cast content to TextContent")
	ErrFailedCastToToolUseContent = fmt.Errorf("failed to cast content to ToolUseContent")
	ErrInvalidFieldType           = fmt.Errorf("invalid field type")
	ErrInvalidContentBlockField   = fmt.Errorf("invalid content block field")
)

type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type messagePayload struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	System      string        `json:"system,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	StopWords   []string      `json:"stop_sequences,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Temperature float64       `json:"temperature"`
	Tools       []Tool        `json:"tools,omitempty"`
	TopP        float64       `json:"top_p,omitempty"`

	StreamingFunc func(ctx context.Context, chunk []byte) error `json:"-"`
}

// Tool used for the request message payload.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

// Content can be TextContent or ToolUseContent depending on the type.
type Content interface {
	GetType() string
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (tc TextContent) GetType() string {
	return tc.Type
}

type ToolUseContent struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`

	// we added this field to the struct, to accumulate the input while in streaming mode
	inputAccumulator string
}

func (tuc ToolUseContent) GetType() string {
	return tuc.Type
}

type ToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

func (trc ToolResultContent) GetType() string {
	return trc.Type
}

type MessageResponsePayload struct {
	Content      []Content `json:"content"`
	ID           string    `json:"id"`
	Model        string    `json:"model"`
	Role         string    `json:"role"`
	StopReason   string    `json:"stop_reason"`
	StopSequence string    `json:"stop_sequence"`
	Type         string    `json:"type"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (m *MessageResponsePayload) UnmarshalJSON(data []byte) error {
	type Alias MessageResponsePayload
	aux := &struct {
		Content []json.RawMessage `json:"content"`
		*Alias
	}{
		Alias: (*Alias)(m),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	for _, raw := range aux.Content {
		var typeStruct struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &typeStruct); err != nil {
			return err
		}

		switch typeStruct.Type {
		case "text":
			tc := &TextContent{}
			if err := json.Unmarshal(raw, tc); err != nil {
				return err
			}
			m.Content = append(m.Content, tc)
		case "tool_use":
			tuc := &ToolUseContent{}
			if err := json.Unmarshal(raw, tuc); err != nil {
				return err
			}
			m.Content = append(m.Content, tuc)
		default:
			return fmt.Errorf("unknown content type: %s\n%v", typeStruct.Type, string(raw))
		}
	}

	return nil
}

func (c *Client) setMessageDefaults(payload *messagePayload) {
	// Set defaults
	if payload.MaxTokens == 0 {
		payload.MaxTokens = 2048
	}

	if len(payload.StopWords) == 0 {
		payload.StopWords = nil
	}

	switch {
	// Prefer the model specified in the payload.
	case payload.Model != "":

	// If no model is set in the payload, take the one specified in the client.
	case c.Model != "":
		payload.Model = c.Model
	// Fallback: use the default model
	default:
		payload.Model = defaultModel
	}
	if payload.StreamingFunc != nil {
		payload.Stream = true
	}
}

func (c *Client) createMessage(ctx context.Context, payload *messagePayload) (*MessageResponsePayload, error) {
	c.setMessageDefaults(payload)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.do(ctx, "/messages", payloadBytes)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.decodeError(resp)
	}

	if payload.StreamingFunc != nil {
		return parseStreamingMessageResponse(ctx, resp, payload)
	}

	var response MessageResponsePayload
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &response, nil
}

type MessageEvent struct {
	Response *MessageResponsePayload
	Err      error
}

func parseStreamingMessageResponse(ctx context.Context, r *http.Response, payload *messagePayload) (*MessageResponsePayload, error) {
	scanner := bufio.NewScanner(r.Body)
	eventChan := make(chan MessageEvent)

	go func() {
		defer close(eventChan)
		var response MessageResponsePayload
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			event, err := parseStreamEvent(data)
			if err != nil {
				eventChan <- MessageEvent{Response: nil, Err: fmt.Errorf("failed to parse stream event: %w", err)}
				return
			}
			response, err = processStreamEvent(ctx, event, payload, response, eventChan)
			if err != nil {
				eventChan <- MessageEvent{Response: nil, Err: fmt.Errorf("failed to process stream event: %w", err)}
				return
			}
		}
		if err := scanner.Err(); err != nil {
			eventChan <- MessageEvent{Response: nil, Err: fmt.Errorf("issue scanning response: %w", err)}
		}
	}()

	var lastResponse *MessageResponsePayload
	for event := range eventChan {
		if event.Err != nil {
			return nil, event.Err
		}
		lastResponse = event.Response
	}

	return lastResponse, nil
}

func parseStreamEvent(data string) (map[string]interface{}, error) {
	var event map[string]interface{}
	err := json.NewDecoder(bytes.NewReader([]byte(data))).Decode(&event)
	return event, err
}

func processStreamEvent(ctx context.Context, event map[string]interface{}, payload *messagePayload, response MessageResponsePayload, eventChan chan<- MessageEvent) (MessageResponsePayload, error) {
	eventType, ok := event["type"].(string)
	if !ok {
		return response, ErrInvalidEventType
	}
	switch eventType {
	case "message_start":
		return handleMessageStartEvent(event, response)
	case "content_block_start":
		return handleContentBlockStartEvent(event, response)
	case "content_block_delta":
		return handleContentBlockDeltaEvent(ctx, event, response, payload)
	case "content_block_stop":
		return handleContentBlockStopEvent(event, response)
	case "message_delta":
		return handleMessageDeltaEvent(event, response)
	case "message_stop":
		eventChan <- MessageEvent{Response: &response, Err: nil}
	case "ping":
		// Nothing to do here
	case "error":
		eventChan <- MessageEvent{Response: nil, Err: fmt.Errorf("received error event: %v", event)}
	default:
		log.Printf("unknown event type: %s - %v", eventType, event)
	}
	return response, nil
}

func handleMessageStartEvent(event map[string]interface{}, response MessageResponsePayload) (MessageResponsePayload, error) {
	message, ok := event["message"].(map[string]interface{})
	if !ok {
		return response, ErrInvalidMessageField
	}

	usage, ok := message["usage"].(map[string]interface{})
	if !ok {
		return response, ErrInvalidUsageField
	}

	inputTokens, err := getFloat64(usage, "input_tokens")
	if err != nil {
		return response, err
	}

	response.ID = getString(message, "id")
	response.Model = getString(message, "model")
	response.Role = getString(message, "role")
	response.Type = getString(message, "type")
	response.Usage.InputTokens = int(inputTokens)

	return response, nil
}

func handleContentBlockStartEvent(event map[string]interface{}, response MessageResponsePayload) (MessageResponsePayload, error) {
	indexValue, ok := event["index"].(float64)
	if !ok {
		return response, ErrInvalidIndexField
	}
	index := int(indexValue)

	contentBlock, ok := event["content_block"].(map[string]any)
	if !ok {
		return response, ErrInvalidContentBlockField
	}
	if index < len(response.Content) {
		return response, fmt.Errorf("expected index to be greater then content")
	}

	contentType, ok := contentBlock["type"].(string)
	if !ok {
		return response, fmt.Errorf("expected content block type to be a string")
	}
	switch contentType {
	case "text":
		response.Content = append(response.Content, &TextContent{
			Type: "text",
			Text: contentBlock["text"].(string),
		})
	case "tool_use":
		response.Content = append(response.Content, &ToolUseContent{
			Type:  "tool_use",
			ID:    contentBlock["id"].(string),
			Name:  contentBlock["name"].(string),
			Input: make(map[string]interface{}),
		})
	}

	return response, nil
}

func handleContentBlockStopEvent(event map[string]interface{}, response MessageResponsePayload) (MessageResponsePayload, error) {
	indexValue, ok := event["index"].(float64)
	if !ok {
		return response, ErrInvalidIndexField
	}
	index := int(indexValue)
	if index >= len(response.Content) {
		return response, fmt.Errorf("expected index to be in content array")
	}

	contentBlock := response.Content[index]
	switch v := contentBlock.(type) {
	case *TextContent:
		// nothing to do
	case *ToolUseContent:
		if err := json.Unmarshal([]byte(v.inputAccumulator), &v.Input); err != nil {
			return response, fmt.Errorf("failed to unmarshal input: %w", err)
		}
		response.Content[index] = v
	}

	return response, nil
}

func handleContentBlockDeltaEvent(ctx context.Context, event map[string]interface{}, response MessageResponsePayload, payload *messagePayload) (MessageResponsePayload, error) {
	indexValue, ok := event["index"].(float64)
	if !ok {
		return response, ErrInvalidIndexField
	}
	index := int(indexValue)

	delta, ok := event["delta"].(map[string]interface{})
	if !ok {
		return response, ErrInvalidDeltaField
	}
	deltaType, ok := delta["type"].(string)
	if !ok {
		return response, ErrInvalidDeltaTypeField
	}

	text := ""

	if deltaType == "text_delta" {
		text, ok = delta["text"].(string)
		if !ok {
			return response, ErrInvalidDeltaTextField
		}
		if len(response.Content) <= index {
			return response, ErrContentIndexOutOfRange
		}
		textContent, ok := response.Content[index].(*TextContent)
		if !ok {
			return response, ErrFailedCastToTextContent
		}
		textContent.Text += text
	}

	if deltaType == "input_json_delta" {
		text, ok = delta["partial_json"].(string)
		if !ok {
			return response, ErrInvalidDeltaInputJSONField
		}
		if len(response.Content) <= index {
			return response, ErrContentIndexOutOfRange
		}
		textContent, ok := response.Content[index].(*ToolUseContent)
		if !ok {
			return response, ErrFailedCastToToolUseContent
		}
		textContent.inputAccumulator += text
	}

	if payload.StreamingFunc != nil {
		err := payload.StreamingFunc(ctx, []byte(text))
		if err != nil {
			return response, fmt.Errorf("streaming func returned an error: %w", err)
		}
	}
	return response, nil
}

func handleMessageDeltaEvent(event map[string]interface{}, response MessageResponsePayload) (MessageResponsePayload, error) {
	delta, ok := event["delta"].(map[string]interface{})
	if !ok {
		return response, ErrInvalidDeltaField
	}
	if stopReason, ok := delta["stop_reason"].(string); ok {
		response.StopReason = stopReason
	}

	usage, ok := event["usage"].(map[string]interface{})
	if !ok {
		return response, ErrInvalidUsageField
	}
	if outputTokens, ok := usage["output_tokens"].(float64); ok {
		response.Usage.OutputTokens = int(outputTokens)
	}
	return response, nil
}

func getString(m map[string]interface{}, key string) string {
	value, ok := m[key].(string)
	if !ok {
		return ""
	}
	return value
}

func getFloat64(m map[string]interface{}, key string) (float64, error) {
	value, ok := m[key].(float64)
	if !ok {
		return 0, ErrInvalidFieldType
	}
	return value, nil
}
