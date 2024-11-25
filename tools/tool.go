package tools

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// Tool is a tool for the llm agent to interact with different applications.
type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, input string) (string, error)
}

var _ Tool = &NativeTool{}

type NativeToolCallFunc[I any, O any] func(context.Context, I) (O, error)

type NativeTool struct {
	name        string
	description string
	call        func(context.Context, string) (string, error)
	jsonSchema  map[string]any
}

func (n *NativeTool) Name() string {
	return n.name
}

func (n *NativeTool) Description() string {
	return n.description
}

func (n *NativeTool) Call(ctx context.Context, input string) (string, error) {
	return n.call(ctx, input)
}

func (n *NativeTool) ToLLmTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        n.name,
			Description: n.description,
			Parameters:  n.jsonSchema,
			Strict:      false,
		},
	}
}
