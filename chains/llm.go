package chains

import (
	"context"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/schema"
)

const _llmChainDefaultOutputKey = "text"

type LLMChain struct {
	Prompt           prompts.FormatPrompter
	LLM              llms.Model
	Memory           schema.Memory
	CallbacksHandler callbacks.Handler
	OutputParser     schema.OutputParser[any]
	// When enabled usesMultiplePrompts will not 'flatten' the prompt into a single message.
	UseMultiPrompt bool

	OutputKey string
}

var (
	_ Chain                  = &LLMChain{}
	_ callbacks.HandlerHaver = &LLMChain{}
)

// NewLLMChain creates a new LLMChain with an LLM and a prompt.
func NewLLMChain(llm llms.Model, prompt prompts.FormatPrompter, opts ...ChainCallOption) *LLMChain {
	opt := &chainCallOption{}
	for _, o := range opts {
		o(opt)
	}
	chain := &LLMChain{
		Prompt:           prompt,
		LLM:              llm,
		OutputParser:     outputparser.NewSimple(),
		Memory:           memory.NewSimple(),
		OutputKey:        _llmChainDefaultOutputKey,
		CallbacksHandler: opt.CallbackHandler,
		UseMultiPrompt:   false,
	}

	return chain
}

// Call formats the prompts with the input values, generates using the llm, and parses
// the output from the llm with the output parser. This function should not be called
// directly, use rather the Call or Run function if the prompt only requires one input
// value.
func (c LLMChain) Call(ctx context.Context, values map[string]any, options ...ChainCallOption) (map[string]any, error) {
	promptValue, err := c.Prompt.FormatPrompt(values)
	if err != nil {
		return nil, err
	}

	var output string
	if c.UseMultiPrompt {
		output, err = llms.GenerateFromMultiPrompt(ctx, c.LLM, chatMessagesToLLmMessageContent(promptValue.Messages()), getLLMCallOptions(options...)...)
	} else {
		output, err = llms.GenerateFromSinglePrompt(ctx, c.LLM, promptValue.String(), getLLMCallOptions(options...)...)
	}
	if err != nil {
		return nil, err
	}

	finalOutput, err := c.OutputParser.ParseWithPrompt(output, promptValue)
	if err != nil {
		return nil, err
	}

	return map[string]any{c.OutputKey: finalOutput}, nil
}

// GetMemory returns the memory.
func (c LLMChain) GetMemory() schema.Memory { //nolint:ireturn
	return c.Memory //nolint:ireturn
}

func (c LLMChain) GetCallbackHandler() callbacks.Handler { //nolint:ireturn
	return c.CallbacksHandler
}

// GetInputKeys returns the expected input keys.
func (c LLMChain) GetInputKeys() []string {
	return append([]string{}, c.Prompt.GetInputVariables()...)
}

// GetOutputKeys returns the output keys the chain will return.
func (c LLMChain) GetOutputKeys() []string {
	return []string{c.OutputKey}
}

// Convert ChatMessage to MessageContent.
// Each ChatMessage is directly converted to a MessageContent with the same content and type.
func chatMessagesToLLmMessageContent(chatMessages []llms.ChatMessage) []llms.MessageContent {
	msgs := make([]llms.MessageContent, len(chatMessages))
	for idx, m := range chatMessages {
		msgs[idx] = chatMessageToLLm(m)
	}
	return msgs
}

func chatMessageToLLm(in llms.ChatMessage) llms.MessageContent {
	return llms.MessageContent{
		Parts: []llms.ContentPart{
			llms.TextContent{Text: in.GetContent()},
		},
		Role: in.GetType(),
	}
}
