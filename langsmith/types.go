package langsmith

import (
	"time"

	"github.com/tmc/langchaingo/llms"
)

type KVMap map[string]any

func valueIfSetOtherwiseNil[T comparable](v T) *T {
	var empty T
	if v == empty {
		return nil
	}

	return &v
}

func timeToMillisecondsPtr(t time.Time) *int64 {
	if t.IsZero() {
		return nil
	}

	return ptr(t.UnixMilli())
}

func ptr[T any](v T) *T {
	return &v
}

type (
	inputs []input
	input  struct {
		Role    string             `json:"role"`
		Content []llms.ContentPart `json:"content"`
	}
)

func inputsFromMessages(ms []llms.MessageContent) inputs {
	inputs := make(inputs, len(ms))
	for i, msg := range ms {
		inputs[i] = input{Role: string(msg.Role), Content: msg.Parts}
	}
	return inputs
}
