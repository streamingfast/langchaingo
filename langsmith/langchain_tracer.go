package langsmith

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

var _ callbacks.Handler = (*LangChainTracer)(nil)

type LangChainTracer struct {
	name        string
	projectName string
	exampleID   string
	client      *Client

	activeTree *RunTree
}

func NewLangChainTracer(opts ...LangChainTracerOption) (*LangChainTracer, error) {
	tracer := &LangChainTracer{
		name:        "langchain_tracer",
		projectName: envOr(os.Getenv("LANGCHAIN_PROJECT"), "default"),
		exampleID:   "",
		client:      nil,
	}

	for _, opt := range opts {
		opt.apply(tracer)
	}

	if tracer.client == nil {
		var err error
		tracer.client, err = NewClient()
		if err != nil {
			return nil, fmt.Errorf("new langsmith client: %w", err)
		}
	}

	return tracer, nil
}

// HandleText implements callbacks.Handler.
func (t *LangChainTracer) HandleText(ctx context.Context, text string) {
	fmt.Println("HandleText", text)
}

// HandleLLMStart implements callbacks.Handler.
func (t *LangChainTracer) HandleLLMStart(ctx context.Context, prompts []string) {
	t.activeTree.SetName("llm_start").SetProjectName(t.projectName).SetRunType("llm").SetClient(t.client).SetInputs(KVMap{
		"prompts": prompts,
	})

	if err := t.activeTree.postRun(ctx, true); err != nil {
		t.logLangSmithError("llm_start", "post run", err)
		return
	}
}

func (t *LangChainTracer) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	fmt.Println("HandleLLMGenerateContentStart", ms)
	t.activeTree.SetClient(t.client).SetEndTime(time.Now()).SetOutputs(KVMap{
		"ms": ms, // fixme: validate these fields in the generate content
	})

	if err := t.activeTree.patchRun(ctx); err != nil {
		t.logLangSmithError("handle_llm_generate_content_start", "patch run", err)
		return
	}
}

func (t *LangChainTracer) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	fmt.Println("HandleLLMGenerateContentEnd", res)
}

// HandleLLMError implements callbacks.Handler.
func (t *LangChainTracer) HandleLLMError(ctx context.Context, err error) {
	t.activeTree.SetError(err.Error()).SetEndTime(time.Now())

	if err := t.activeTree.patchRun(ctx); err != nil {
		t.logLangSmithError("llm_error", "patch run", err)
		return
	}

	t.activeTree = nil
}

// HandleChainStart implements callbacks.Handler.
func (t *LangChainTracer) HandleChainStart(ctx context.Context, inputs map[string]any) {
	fmt.Println("HandleChainStart")
	t.activeTree = NewRunTree().SetName("chain_start").SetClient(t.client).SetProjectName(t.projectName).SetRunType("llm").SetInputs(KVMap{
		"inputs": inputs,
	})

	if err := t.activeTree.postRun(ctx, true); err != nil {
		t.logLangSmithError("handle_chain_start", "post run", err)
		return
	}
}

// HandleChainEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	fmt.Println("HandleChainEnd", outputs)
	t.activeTree = NewRunTree().SetName("chain_end").SetClient(t.client).SetProjectName(t.projectName).SetRunType("llm").SetOutputs(KVMap{
		"output": outputs,
	}).SetEndTime(time.Now())

	fmt.Println(t.activeTree == nil)
	if err := t.activeTree.postRun(ctx, true); err != nil { // FIXME: this should be a patchRun instead to update the same uuid that started the chain
		t.logLangSmithError("handle_chain_end", "patch run", err)
		return
	}

	t.activeTree = nil
}

// HandleChainError implements callbacks.Handler.
func (t *LangChainTracer) HandleChainError(ctx context.Context, err error) {
	fmt.Println("HandleChainError", err)
}

// HandleToolStart implements callbacks.Handler.
func (t *LangChainTracer) HandleToolStart(ctx context.Context, input string) {
	fmt.Println("HandleToolStart", input)
}

// HandleToolEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleToolEnd(ctx context.Context, output string) {
	fmt.Println("HandleToolEnd", output)
}

// HandleToolError implements callbacks.Handler.
func (t *LangChainTracer) HandleToolError(ctx context.Context, err error) {
	fmt.Println("HandleToolError", err)
}

// HandleAgentAction implements callbacks.Handler.
func (t *LangChainTracer) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	fmt.Println("HandleAgentAction", action)
}

// HandleAgentFinish implements callbacks.Handler.
func (t *LangChainTracer) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	fmt.Println("HandleAgentFinish", finish)
}

// HandleRetrieverStart implements callbacks.Handler.
func (t *LangChainTracer) HandleRetrieverStart(ctx context.Context, query string) {
	fmt.Println("HandleRetrieverStart", query)
}

// HandleRetrieverEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	fmt.Println("HandleRetrieverEnd", query, documents)
}

// HandleStreamingFunc implements callbacks.Handler.
func (t *LangChainTracer) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	// do nothing
}

func (t *LangChainTracer) logLangSmithError(handlerName string, tag string, err error) {
	fmt.Fprintf(os.Stderr, "We were not able to %s to LangSmith server via handler %q: %s", handlerName, tag, err)
}
