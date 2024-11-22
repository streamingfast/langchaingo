package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/streamingfast/logging"
	"github.com/tmc/langchaingo/examples/llm-chain-full-example/ai"
	aiconfig "github.com/tmc/langchaingo/examples/llm-chain-full-example/ai/config"
)

var flagLLMModel = flag.String("llm-model", "gpt-4o-mini", "model to use (e.g. 'gpt-4o', 'gpt-4o-mini', 'claude-3-5-haiku', 'claude-3-5-sonnet')")
var flagOpenaiAPIKey = flag.String("openai-api-key", "", "OpenAI API key")
var flagOpenaiOrganization = flag.String("openai-organization", "", "OpenAI organization")
var flagAnthropicAPIKey = flag.String("anthropic-api-key", "", "Anthropic API Key")
var flagLangchainAPIKey = flag.String("langchain-api-key", "", "Langchain API Key")
var flagLangchainProject = flag.String("langchain-project", "", "Langchain Project")

var logger, _ = logging.PackageLogger("llm_chain_full_example", "github.com/tmc/langchaingo/examples/llm-chain-full-example")

func init() {
	logging.InstantiateLoggers()
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	if flagLLMModel == nil {
		return fmt.Errorf("llm model must be provided")
	}

	llmModel, err := ai.NewLLMModel(*flagLLMModel)
	if err != nil {
		return fmt.Errorf("llm model: %w", err)
	}

	aiClient, err := setupAiClient()
	if err != nil {
		return err
	}

	schemaPrompt := `
Answer in a JSON format that respects the following JSON schema.
When the JSON schema specifies an enum list for a string, ensure that the returned value is actually in the list of allowed values, defined of the "enum" field and mention the available value in the 'chain_of_thought' field of your response.

{
  "type": "object",
  "properties": {
    "chain-of-thought": {
      "type": "string",
      "description": "Explanation of the chain of thought leading to the question being answered"
    },
    "answer": {
      "type": "string",
      "description": "answer the question",
    },
    "confidence-score": {
      "type": "number",
      "description": "Confidence score of the answer. It should be between 0 and 1"
    }
  },
  "required": ["chain_of_thought", "answer", "confidence-score"],
  "additionalProperties": false
}

Here is an example of a valid JSON output

{"chain-of-thought":"To determine the allocations, I considered the deployment details and timestamps provided, ensuring the tokens are allocated proportionally. Confidence was derived based on the consistency of the data.","answer":"this is the answer","confidence-score":0.92}	
`

	prompts := &ai.Prompt{
		Model:      llmModel,
		PromptTmpl: "You are a translation expert",
		SchemaTmpl: schemaPrompt,
		HumanTmpl:  "Translate the following text from {{.inputLanguage}} to {{.outputLanguage}}. {{.text}}",
		Vars:       make(ai.Variable).WithVariable("inputLanguage", "English").WithVariable("outputLanguage", "French").WithVariable("text", "Hello, how are you?"),
	}

	runID := uuid.New().String()
	fmt.Println("> Running prompt with runID", runID)

	out, err := aiClient.RunPrompt(ctx, "example", prompts, runID, logger)
	if err != nil {
		return fmt.Errorf("run prompt: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func getFlagOrEnv(flagValue *string, envName string) string {
	if flagValue == nil {
		return os.Getenv(envName)
	}
	if *flagValue == "" {
		return os.Getenv(envName)
	}
	return *flagValue
}

func setupAiClient() (*ai.Client, error) {
	var llmCfg aiconfig.LLMConfig

	if openAIKey := getFlagOrEnv(flagOpenaiAPIKey, "OPENAI_API_KEY"); openAIKey != "" {
		llmCfg = aiconfig.NewOpenAIConfig(openAIKey, getFlagOrEnv(flagOpenaiOrganization, "OPENAI_ORGANIZATION"))
	} else if anthropicApiKey := getFlagOrEnv(flagAnthropicAPIKey, "ANTHROPIC_API_KEY"); anthropicApiKey != "" {
		llmCfg = aiconfig.NewAnthropicConfig(anthropicApiKey)
	} else {
		return nil, fmt.Errorf("no LLM config provided")
	}

	aiCfg := aiconfig.NewConfig(
		llmCfg,
		getFlagOrEnv(flagLangchainProject, "LANGCHAIN_PROJECT"),
		getFlagOrEnv(flagLangchainAPIKey, "LANGCHAIN_API_KEY"),
	)

	llm, err := ai.New(aiCfg, logger)
	if err != nil {
		return nil, fmt.Errorf("ai client: %w", err)
	}

	return llm, nil
}
