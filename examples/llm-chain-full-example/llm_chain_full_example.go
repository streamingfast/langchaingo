package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/streamingfast/logging"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/langsmith"
)

var flagLLMModel = flag.String("llm-model", "gpt-4o-mini", "model to use (e.g. 'gpt-4o', 'gpt-4o-mini', 'claude-3-5-haiku', 'claude-3-5-sonnet')")

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

	llmModel, err := getLLMModel()
	if err != nil {
		return fmt.Errorf("llm model: %w", err)
	}

	llmChain := chains.NewLLMChainV2(llmModel, getPromptTemplate())
	llmChain.RegisterTools(getCurrentWeatherToolCall, getStockPriceToolCall)

	langsmithClient, err := langsmith.NewClient(
		langsmith.WithAPIKey(os.Getenv("LANGCHAIN_API_KEY")),
		langsmith.WithAPIURL("https://api.smith.langchain.com"),
		langsmith.WithClientLogger(&LangchainLogger{logger}),
	)
	if err != nil {
		return fmt.Errorf("new langsmith client: %w", err)
	}

	langchainProject := os.Getenv("LANGCHAIN_PROJECT")

	// ----------------------------------------------------------------------------
	// ----------------------------------------------------------------------------
	// --- This would happen on every RUN ---
	runID := uuid.New().String()
	langChainTracer, err := langsmith.NewTracer(
		langsmith.WithLogger(&LangchainLogger{Logger: logger}),
		langsmith.WithProjectName(langchainProject),
		langsmith.WithClient(langsmithClient),
		langsmith.WithRunID(runID),
	)
	if err != nil {
		return fmt.Errorf("chain tracer: %w", err)
	}

	fmt.Println("> Running prompt with runID", runID)
	out, err := chains.Call(
		ctx,
		llmChain,
		map[string]any{
			"location": "Montreal, QC",
		},
		chains.WithModel(*flagLLMModel),
		chains.WithTemperature(0.1),
		chains.WithSeed(123),
		chains.WithCallback(langChainTracer),
	)
	if err != nil {
		return err
	}

	response := out["text"].(string)

	var output Response
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		return fmt.Errorf("unmarshal output: %w", err)
	}

	fmt.Println("")
	fmt.Println("------------------------------------------------------")
	fmt.Println("Chain of thought: ")
	fmt.Println(output.ChainOfThought)
	fmt.Println("------------------------------------------------------")
	fmt.Println("> Answer: ", output.Answer)
	fmt.Println("> Confidence score: ", output.Confidence)
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

type Response struct {
	ChainOfThought string  `json:"chain-of-thought"`
	Answer         string  `json:"answer"`
	Confidence     float64 `json:"confidence-score"`
}
