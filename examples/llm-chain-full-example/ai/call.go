package ai

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/langsmith"
	"github.com/tmc/langchaingo/prompts"
	"go.uber.org/zap"
)

const MAX_RETRIES = 3
const SEED = 123

func (c *Client) langsmithTracer(runId string, logger *zap.Logger) (*langsmith.LangChainTracer, error) {
	langChainTracer, err := langsmith.NewTracer(
		langsmith.WithLogger(&LangchainLogger{Logger: logger}),
		langsmith.WithProjectName(c.langsmithProjectName),
		langsmith.WithClient(c.langsmithClient),
		langsmith.WithRunID(runId),
	)
	if err != nil {
		return nil, fmt.Errorf("chain tracer: %w", err)
	}
	return langChainTracer, nil
}

func (c *Client) call(ctx context.Context, runId string, template prompts.FormatPrompter, values map[string]any, llmModel LLMModel, logger *zap.Logger) (string, error) {
	var output map[string]any

	langsmithTracer, err := c.langsmithTracer(runId, logger)
	if err != nil {
		return "", fmt.Errorf("langsmith tracer: %w", err)
	}

	llmChain := chains.NewLLMChain(c.llmModel, template, chains.WithCallback(langsmithTracer))
	llmChain.UseMultiPrompt = true

	output, err = chains.Call(ctx, llmChain, values,
		chains.WithModel(llmModel.String()),
		chains.WithTemperature(0.1),
		chains.WithSeed(SEED),
		chains.WithMaxTokens(8192),
	)
	if err != nil {
		return "", fmt.Errorf("llm chain: %w", err)
	}

	contentRaw := output["text"]
	content := ""
	if contentRaw != nil {
		content = contentRaw.(string)
	}

	return content, nil
}
