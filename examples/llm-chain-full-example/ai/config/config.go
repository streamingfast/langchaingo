package ai

import (
	"github.com/streamingfast/dstore"
)

type LLMConfig interface {
	IsLLMConfig()
}

type Config struct {
	llmConfig            LLMConfig
	langchainProjectName string
	langchainApiKey      string
	debugFileStore       dstore.Store
}

func NewConfig(llmConfig LLMConfig, langchainProjectName, langchainApiKey string) *Config {
	return &Config{
		llmConfig:            llmConfig,
		langchainProjectName: langchainProjectName,
		langchainApiKey:      langchainApiKey,
	}
}

func (c *Config) WithDebugStore(debugFileStore dstore.Store) *Config {
	c.debugFileStore = debugFileStore
	return c
}

func (c *Config) GetLangchain() (apiKey string, projectName string) {
	return c.langchainApiKey, c.langchainProjectName
}

func (c *Config) GetDebugStore() dstore.Store {
	return c.debugFileStore
}

func (c *Config) GetLLMConfig() LLMConfig {
	return c.llmConfig
}

type OpenAIConfig struct {
	ApiKey         string
	OrganizationID string
}

func NewOpenAIConfig(apiKey, organizationID string) LLMConfig {
	return &OpenAIConfig{
		ApiKey:         apiKey,
		OrganizationID: organizationID,
	}
}

func (c *OpenAIConfig) IsLLMConfig() {}

type AnthropicConfig struct {
	ApiKey string
}

func NewAnthropicConfig(apiKey string) LLMConfig {
	return &AnthropicConfig{
		ApiKey: apiKey,
	}
}

func (c *AnthropicConfig) IsLLMConfig() {}
