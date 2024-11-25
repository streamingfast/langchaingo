package chains

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tools"
)

type LLMChainV2 struct {
	prompt       prompts.FormatPrompter
	llmModel     llms.Model
	memory       schema.Memory
	tools        map[string]*tools.NativeTool
	outputParser schema.OutputParser[any]
}

var (
	_ Chain                  = &LLMChain{}
	_ callbacks.HandlerHaver = &LLMChain{}
)

// NewLLMChain creates a new LLMChain with an LLM and a prompt.
func NewLLMChainV2(llm llms.Model, prompt prompts.FormatPrompter) *LLMChainV2 {
	return &LLMChainV2{
		prompt:       prompt,
		llmModel:     llm,
		tools:        make(map[string]*tools.NativeTool),
		outputParser: outputparser.NewSimple(),
		memory:       memory.NewSimple(),
	}
}

func (c LLMChainV2) RegisterTools(tools ...*tools.NativeTool) error {
	for _, tool := range tools {
		if _, found := c.tools[tool.Name()]; found {
			return fmt.Errorf("tool already registered: %s", tool.Name())
		}
		c.tools[tool.Name()] = tool
	}
	return nil
}
func (c LLMChainV2) getTools() (out []llms.Tool) {
	for _, tool := range c.tools {
		out = append(out, tool.ToLLmTool())
	}
	return out

}

// Call formats the prompts with the input values, generates using the llm, and parses
// the output from the llm with the output parser. This function should not be called
// directly, use rather the Call or Run function if the prompt only requires one input
// value.
func (c LLMChainV2) Call(ctx context.Context, values map[string]any, options ...ChainCallOption) (map[string]any, error) {
	promptValue, err := c.prompt.FormatPrompt(values)
	if err != nil {
		return nil, err
	}

	messages := ChatMessagesToLLmMessageContent(promptValue.Messages())
	llmsOptions := getLLMCallOptions(options...)
	llmsOptions = append(llmsOptions, llms.WithTools(c.getTools()))
	out, err := c.call(ctx, messages, llmsOptions...)
	if err != nil {
		return nil, err
	}

	return map[string]any{"text": string(out)}, nil
}

func (c LLMChainV2) call(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) ([]byte, error) {
	count := 0
	for {
		if count > 5 {
			return nil, fmt.Errorf("too many iterations")
		}

		llmResponse, err := c.llmModel.GenerateContent(ctx, messages, options...)
		if err != nil {
			return nil, fmt.Errorf("llm generate content: %w", err)
		}
		if llmResponse == nil || llmResponse.Choices == nil {
			return nil, fmt.Errorf("content response is nil")
		}
		if len(llmResponse.Choices) == 0 {
			return nil, fmt.Errorf("empty response from model")
		}

		toolCalls := getToolCalls(llmResponse)
		if len(toolCalls) == 0 {
			return []byte(llmResponse.Choices[0].Content), nil
		}

		toolMessages, err := c.runTools(ctx, toolCalls)
		if err != nil {
			return nil, fmt.Errorf("run tools: %w", err)
		}

		messages = append(messages, toolMessages...)
		count++
	}
}

// GetMemory returns the memory.
func (c LLMChainV2) GetMemory() schema.Memory { //nolint:ireturn
	return c.memory //nolint:ireturn
}

// GetInputKeys returns the expected input keys.
func (c LLMChainV2) GetInputKeys() []string {
	return append([]string{}, c.prompt.GetInputVariables()...)
}

// GetOutputKeys returns the output keys the chain will return.
func (c LLMChainV2) GetOutputKeys() []string {
	return []string{"text"}
}

func (r *LLMChainV2) runTools(ctx context.Context, toolCalls []llms.ToolCall) (out []llms.MessageContent, err error) {
	for _, toolCall := range toolCalls {
		tcall, found := r.tools[toolCall.FunctionCall.Name]
		if !found {
			return nil, fmt.Errorf("tool not found: %s", tcall.Name())
		}

		response, err := tcall.Call(ctx, toolCall.FunctionCall.Arguments)
		if err != nil {
			return nil, fmt.Errorf("failed tool %s: %w", toolCall.FunctionCall, err)
		}
		out = append(out, llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
			Parts: []llms.ContentPart{
				llms.ToolCall{
					ID:   toolCall.ID,
					Type: toolCall.Type,
					FunctionCall: &llms.FunctionCall{
						Name:      toolCall.FunctionCall.Name,
						Arguments: toolCall.FunctionCall.Arguments,
					},
				},
			},
		}, llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.ToolCallResponse{
					ToolCallID: toolCall.ID,
					Name:       toolCall.FunctionCall.Name,
					Content:    response,
				},
			},
		})
	}
	return out, nil

}

func getToolCalls(contentResponse *llms.ContentResponse) (out []llms.ToolCall) {
	for _, choice := range contentResponse.Choices {
		if choice.StopReason == "tool_calls" {
			out = append(out, choice.ToolCalls...)
		}
	}

	return out
}
