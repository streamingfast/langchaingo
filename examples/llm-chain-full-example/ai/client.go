package ai

import (
	"fmt"

	"github.com/streamingfast/dstore"
	aiconfig "github.com/tmc/langchaingo/examples/llm-chain-full-example/ai/config"
	"github.com/tmc/langchaingo/langsmith"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
)

type Client struct {
	// langsmith configuration
	langsmithClient      *langsmith.Client
	langsmithProjectName string

	// llmModel is the language model used for generating text
	// we are using OpenAI but it can change in the guture
	llmModel llms.Model

	debugFileStore dstore.Store
}

func New(cfg *aiconfig.Config, logger *zap.Logger) (*Client, error) {

	langchainApiKey, langchainProject := cfg.GetLangchain()

	langsmithClient, err := langsmith.NewClient(
		langsmith.WithAPIKey(langchainApiKey),
		langsmith.WithAPIURL("https://api.smith.langchain.com"),
		langsmith.WithClientLogger(&LangchainLogger{logger}),
	)
	if err != nil {
		return nil, fmt.Errorf("new langsmith client: %w", err)
	}

	c := &Client{
		langsmithClient:      langsmithClient,
		langsmithProjectName: langchainProject,
	}
	llmConfig := cfg.GetLLMConfig()
	if llmConfig == nil {
		return nil, fmt.Errorf("llm config is required")
	}

	switch v := llmConfig.(type) {
	case *aiconfig.OpenAIConfig:
		logger.Info("using openai", zap.String("organization", v.OrganizationID))
		c.llmModel, err = openai.New(
			openai.WithToken(v.ApiKey),
			openai.WithOrganization(v.OrganizationID),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to create openai model: %w", err)
		}
	case *aiconfig.AnthropicConfig:
		logger.Info("using anthropic")
		c.llmModel, err = anthropic.New(anthropic.WithToken(v.ApiKey))
		if err != nil {
			return nil, fmt.Errorf("unable to create anthropic model: %w", err)
		}
	}

	if debugFileStore := cfg.GetDebugStore(); debugFileStore != nil {
		c.debugFileStore = debugFileStore
	}
	return c, nil
}
