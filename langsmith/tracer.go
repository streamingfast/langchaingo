package langsmith

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/tracing"
)

var _ callbacks.Handler = (*LangChainTracer)(nil)

type LangChainTracer struct {
	name        string
	projectName string
	client      *Client

	runID       string
	activeTree  *RunTree
	treeStack   stack
	extras      KVMap
	llmMetadata *tracing.TracerLLMMetadata
	logger      LeveledLoggerInterface
}

func NewTracer(opts ...LangChainTracerOption) (*LangChainTracer, error) {
	tracer := &LangChainTracer{
		name:        "RunnableSequence",
		projectName: "default",
		client:      nil,
		runID:       uuid.New().String(),
		logger:      &NopLogger{},
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

func (t *LangChainTracer) GetRunID() string {
	return t.runID
}

func (t *LangChainTracer) resetActiveTree() {
	t.activeTree = nil
}

// HandleText implements callbacks.Handler.
func (t *LangChainTracer) HandleText(_ context.Context, _ string) {
}

func (t *LangChainTracer) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	childTree := t.activeTree.CreateChild()

	t.treeStack = t.treeStack.Push(childTree)

	childTree.
		SetName(t.getSpanName("LLMGenerateContent")).
		SetRunType("llm").
		SetInputs(KVMap{
			"messages": inputsFromMessages(ms),
		}).
		SetExtra(t.getExtra(nil))

	t.activeTree.AppendChild(childTree)

	// Start the run
	if err := childTree.postRun(ctx, true); err != nil {
		t.logLangSmithError("llm_start", "post run", err)
		return
	}
}

func (t *LangChainTracer) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	var childTree *RunTree
	t.treeStack, childTree = t.treeStack.Pop()

	childTree.
		SetOutputs(KVMap{
			"choices": res.Choices,
		})

	if tracingOutput := res.GetTracingOutput(); tracingOutput != nil {
		childTree.SetOutputs(tracingOutput.Output)
	}

	// Close the run
	if err := childTree.patchRun(ctx); err != nil {
		t.logLangSmithError("llm_start", "post run", err)
		return
	}
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
	t.activeTree = NewRunTree(t.runID).
		SetName(t.name).
		SetClient(t.client).
		SetProjectName(t.projectName).
		SetRunType("chain").
		SetInputs(inputs).
		SetExtra(t.getExtra(nil))

	if err := t.activeTree.postRun(ctx, true); err != nil {
		t.logLangSmithError("handle_chain_start", "post run", err)
		return
	}
}

// HandleChainEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	t.activeTree.
		SetOutputs(outputs).
		SetEndTime(time.Now())

	if err := t.activeTree.patchRun(ctx); err != nil {
		t.logLangSmithError("handle_chain_end", "patch run", err)
		return
	}

	t.resetActiveTree()
}

// HandleChainError implements callbacks.Handler.
func (t *LangChainTracer) HandleChainError(ctx context.Context, err error) {
	t.activeTree.SetError(err.Error()).SetEndTime(time.Now())

	if err := t.activeTree.patchRun(ctx); err != nil {
		t.logLangSmithError("handle_chain_error", "patch run", err)
		return
	}

	t.activeTree = nil
}

func (t *LangChainTracer) HandleLLMToolCallStart(ctx context.Context, toolCall llms.ToolCall) {
	childTree := t.activeTree.CreateChild()

	t.treeStack = t.treeStack.Push(childTree)

	childTree.
		SetName(toolCall.FunctionCall.Name).
		SetRunType("tool").
		SetInputs(KVMap{
			"input": toolCall.FunctionCall.Arguments,
		}).
		SetExtra(t.getExtra(KVMap{
			"tool_name": toolCall.FunctionCall.Name,
			"call_id":   toolCall.ID,
		}))

	t.activeTree.AppendChild(childTree)

	// Start the run
	if err := childTree.postRun(ctx, true); err != nil {
		t.logLangSmithError("llm_start", "post run", err)
		return
	}
}

func (t *LangChainTracer) HandleLLMToolCallEnd(ctx context.Context, output string) {
	var childTree *RunTree
	t.treeStack, childTree = t.treeStack.Pop()

	childTree.
		SetOutputs(KVMap{
			"output": output,
		})

	// Close the run
	if err := childTree.patchRun(ctx); err != nil {
		t.logLangSmithError("llm_start", "post run", err)
		return
	}
}

// HandleToolStart implements callbacks.Handler.
func (t *LangChainTracer) HandleToolStart(_ context.Context, input string) {
	t.logger.Debugf("handle tool start: input: %s", input)
}

// HandleToolEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleToolEnd(_ context.Context, output string) {
	t.logger.Debugf("handle tool end: output: %s", output)
}

// HandleToolError implements callbacks.Handler.
func (t *LangChainTracer) HandleToolError(_ context.Context, err error) {
	t.logger.Warnf("handle tool error: %s", err)
}

// HandleAgentAction implements callbacks.Handler.
func (t *LangChainTracer) HandleAgentAction(_ context.Context, action schema.AgentAction) {
	t.logger.Debugf("handle agent action, action: %v", action)
}

// HandleAgentFinish implements callbacks.Handler.
func (t *LangChainTracer) HandleAgentFinish(_ context.Context, finish schema.AgentFinish) {
	t.logger.Debugf("handle agent finish, finish: %v", finish)
}

// HandleRetrieverStart implements callbacks.Handler.
func (t *LangChainTracer) HandleRetrieverStart(_ context.Context, query string) {
	t.logger.Debugf("handle retriever start, query: %s, documents: %v", query)
}

// HandleRetrieverEnd implements callbacks.Handler.
func (t *LangChainTracer) HandleRetrieverEnd(_ context.Context, query string, documents []schema.Document) {
	t.logger.Debugf("handle retriever end, query: %s, documents: %v", query, documents)
}

// HandleStreamingFunc implements callbacks.Handler.
func (t *LangChainTracer) HandleStreamingFunc(_ context.Context, _ []byte) {
	// do nothing
}

func (t *LangChainTracer) logLangSmithError(handlerName string, tag string, err error) {
	t.logger.Debugf("we were not able to %s to LangSmith server via handler %q: %s", handlerName, tag, err)
}

func (t *LangChainTracer) getExtra(others KVMap) KVMap {
	out := t.extras
	if out == nil {
		out = make(KVMap)
	}

	if t.llmMetadata != nil {
		out["metadata"] = KVMap{
			"ls_method":     "traceable",
			"ls_model_name": t.llmMetadata.ModelName,
			"ls_model_type": t.llmMetadata.ModelType,
			"ls_provider":   t.llmMetadata.Provider,
		}
	}

	out["runtime"] = KVMap{
		"runtime": "go",
		"sdk":     "langchaingo",
	}

	for k, v := range others {
		out[k] = v
	}
	return out
}

func (t *LangChainTracer) SetLLMMetadata(metadata *tracing.TracerLLMMetadata) {
	t.llmMetadata = metadata
}

func (t *LangChainTracer) getSpanName(defaultValue string) string {
	if t.llmMetadata != nil {
		return t.llmMetadata.SpanName
	}
	return defaultValue
}
