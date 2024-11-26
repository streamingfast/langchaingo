package chains

import (
	"context"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
)

// ChainCallOption is a function that can be used to modify the behavior of the Call function.
type ChainCallOption func(*chainCallOption)

// For issue #626, each field here has a boolean "set" flag so we can
// distinguish between the case where the option was actually set explicitly
// on chainCallOption, or asked to remain default. The reason we need this is
// that in translating options from ChainCallOption to llms.CallOption, the
// notion of "default value the user didn't explicitly ask to change" is
// violated.
// These flags are hopefully a temporary backwards-compatible solution, until
// we find a more fundamental solution for #626.
type chainCallOption struct {
	// model is the model to use in an LLM call.
	model *string

	// maxTokens is the maximum number of tokens to generate to use in an LLM call.
	maxTokens *int

	// temperature is the temperature for sampling to use in an LLM call, between 0 and 1.
	temperature *float64

	// stopWords is a list of words to stop on to use in an LLM call.
	stopWords    []string
	stopWordsSet bool

	// streamingFunc is a function to be called for each chunk of a streaming response.
	// Return an error to stop streaming early.
	streamingFunc func(ctx context.Context, chunk []byte) error

	// topK is the number of tokens to consider for top-k sampling in an LLM call.
	topK *int

	// topP is the cumulative probability for top-p sampling in an LLM call.
	topP *float64

	// seed is a seed for deterministic sampling in an LLM call.
	seed *int

	// minLength is the minimum length of the generated text in an LLM call.
	minLength *int

	// maxLength is the maximum length of the generated text in an LLM call.
	maxLength *int

	// repetitionPenalty is the repetition penalty for sampling in an LLM call.
	repetitionPenalty *float64

	// CallbackHandler is the callback handler for Chain
	CallbackHandler callbacks.Handler

	// List of tools to pass down
	tools []llms.Tool
}

// WithModel is an option for LLM.Call.
func WithModel(model string) ChainCallOption {
	return func(o *chainCallOption) {
		o.model = &model
	}
}

// WithMaxTokens is an option for LLM.Call.
func WithMaxTokens(maxTokens int) ChainCallOption {
	return func(o *chainCallOption) {
		o.maxTokens = &maxTokens
	}
}

// WithTemperature is an option for LLM.Call.
func WithTemperature(temperature float64) ChainCallOption {
	return func(o *chainCallOption) {
		o.temperature = &temperature
	}
}

// WithStreamingFunc is an option for LLM.Call that allows streaming responses.
func WithStreamingFunc(streamingFunc func(ctx context.Context, chunk []byte) error) ChainCallOption {
	return func(o *chainCallOption) {
		o.streamingFunc = streamingFunc
	}
}

// WithTopK will add an option to use top-k sampling for LLM.Call.
func WithTopK(topK int) ChainCallOption {
	return func(o *chainCallOption) {
		o.topK = &topK
	}
}

// WithTopP	will add an option to use top-p sampling for LLM.Call.
func WithTopP(topP float64) ChainCallOption {
	return func(o *chainCallOption) {
		o.topP = &topP
	}
}

// WithSeed will add an option to use deterministic sampling for LLM.Call.
func WithSeed(seed int) ChainCallOption {
	return func(o *chainCallOption) {
		o.seed = &seed
	}
}

// WithMinLength will add an option to set the minimum length of the generated text for LLM.Call.
func WithMinLength(minLength int) ChainCallOption {
	return func(o *chainCallOption) {
		o.minLength = &minLength
	}
}

// WithMaxLength will add an option to set the maximum length of the generated text for LLM.Call.
func WithMaxLength(maxLength int) ChainCallOption {
	return func(o *chainCallOption) {
		o.maxLength = &maxLength
	}
}

// WithRepetitionPenalty will add an option to set the repetition penalty for sampling.
func WithRepetitionPenalty(repetitionPenalty float64) ChainCallOption {
	return func(o *chainCallOption) {
		o.repetitionPenalty = &repetitionPenalty
	}
}

// WithStopWords is an option for setting the stop words for LLM.Call.
func WithStopWords(stopWords []string) ChainCallOption {
	return func(o *chainCallOption) {
		o.stopWords = stopWords
		o.stopWordsSet = true
	}
}

// WithCallback allows setting a custom Callback Handler.
func WithCallback(callbackHandler callbacks.Handler) ChainCallOption {
	return func(o *chainCallOption) {
		o.CallbackHandler = callbackHandler
	}
}

func withLLmsCallOption[T any](options []llms.CallOption, option *T, applier func(T) llms.CallOption) []llms.CallOption {
	if option != nil {
		options = append(options, applier(*option))
	}
	return options
}

// WithMaxTokens is an option for LLM.Call.
func WithTools(tools []llms.Tool) ChainCallOption {
	return func(o *chainCallOption) {
		o.tools = tools
	}
}

func ChainCallOptionToLLMCallOption(options ...ChainCallOption) []llms.CallOption {
	return getLLMCallOptions(options...)
}

func getLLMCallOptions(options ...ChainCallOption) []llms.CallOption { //nolint:cyclop
	opts := &chainCallOption{}
	for _, option := range options {
		option(opts)
	}
	if opts.streamingFunc == nil && opts.CallbackHandler != nil {
		opts.streamingFunc = func(ctx context.Context, chunk []byte) error {
			opts.CallbackHandler.HandleStreamingFunc(ctx, chunk)
			return nil
		}
	}

	var chainCallOption []llms.CallOption
	chainCallOption = withLLmsCallOption(chainCallOption, opts.model, llms.WithModel)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.maxTokens, llms.WithMaxTokens)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.temperature, llms.WithTemperature)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.topK, llms.WithTopK)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.topP, llms.WithTopP)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.seed, llms.WithSeed)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.minLength, llms.WithMinLength)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.maxLength, llms.WithMaxLength)
	chainCallOption = withLLmsCallOption(chainCallOption, opts.repetitionPenalty, llms.WithRepetitionPenalty)

	if opts.stopWordsSet {
		chainCallOption = append(chainCallOption, llms.WithStopWords(opts.stopWords))
	}

	chainCallOption = append(chainCallOption, llms.WithTools(opts.tools))
	chainCallOption = append(chainCallOption, llms.WithStreamingFunc(opts.streamingFunc))
	return chainCallOption
}

func getChainCallCallbackHandler(options []ChainCallOption) callbacks.Handler {
	opts := &chainCallOption{}
	for _, option := range options {
		option(opts)
	}
	return opts.CallbackHandler
}
