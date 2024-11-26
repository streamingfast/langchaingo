package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/google/uuid"
	"github.com/invopop/jsonschema"
	"github.com/streamingfast/logging"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/langsmith"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
	"github.com/tmc/langchaingo/tools"
)

var flagLLMModel = flag.String("llm-model", "claude-3-5-sonnet-20241022", "model to use (e.g. 'gpt-4o', 'gpt-4o-mini', 'claude-3-5-haiku', 'claude-3-5-sonnet')")

var logger, _ = logging.PackageLogger("llm_chain_full_example", "github.com/tmc/langchaingo/examples/llm-chain-full-example")

func init() {
	logging.InstantiateLoggers()
}

func main() {
	logger.Info("Starting LLM chain full example")
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

	myTools := &Tools{
		logger: logger,
	}

	getCurrentWeatherToolCall, err := tools.NewNativeTool(myTools.getCurrentWeather, "Get a location's current weather")
	if err != nil {
		return fmt.Errorf("getCurrentWeatherToolCall: %w", err)
	}

	getStockPriceToolCall, err := tools.NewNativeTool(myTools.getStockPrice, "Get a symbol's stock price")
	if err != nil {
		return fmt.Errorf("getCurrentWeatherToolCall: %w", err)
	}

	storeRecordToolCall, err := tools.NewNativeTool(myTools.storeRecord, "Store the temprature and stock price")
	if err != nil {
		return fmt.Errorf("storeRecordToolCall: %w", err)
	}

	langsmithClient, err := langsmith.NewClient(
		langsmith.WithAPIKey(os.Getenv("LANGCHAIN_API_KEY")),
		langsmith.WithAPIURL("https://api.smith.langchain.com"),
		langsmith.WithClientLogger(&LangchainLogger{logger}),
	)
	if err != nil {
		return fmt.Errorf("new langsmith client: %w", err)
	}

	fmt.Println("")
	fmt.Println("------------------------------------------------------")
	fmt.Println("Demoing JSON output leveraging a prompt schema")
	if err := runOutputAsJson(ctx, llmModel, []*tools.NativeTool{getCurrentWeatherToolCall, getStockPriceToolCall, storeRecordToolCall}, langsmithClient, os.Getenv("LANGCHAIN_PROJECT")); err != nil {
		return fmt.Errorf("runOutputAsJson: %w", err)
	}

	fmt.Println("")
	fmt.Println("------------------------------------------------------")
	fmt.Println("Demoing JSON output via a tool call")
	if err := runOutputViaTool(ctx, llmModel, []*tools.NativeTool{getCurrentWeatherToolCall, getStockPriceToolCall, storeRecordToolCall}, langsmithClient, os.Getenv("LANGCHAIN_PROJECT")); err != nil {
		return fmt.Errorf("runOutputViaTool: %w", err)
	}

	return nil
}

func runOutputAsJson(ctx context.Context, llmModel llms.Model, tools []*tools.NativeTool, langsmithClient *langsmith.Client, langchainProject string) error {
	prompTemplates := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("You are a weather and stock expert", nil),
		prompts.NewSystemMessagePromptTemplate(schemaPrompt(), nil),
		prompts.NewHumanMessagePromptTemplate("What is the current weather in {{.location}} and the stock price of {{.symbol}}", nil),
	})

	llmChain := chains.NewLLMChainV2(llmModel, prompTemplates)
	llmChain.RegisterTools(tools...)

	// ----------------------------------------------------------------------------
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
			"symbol":   "AAPL",
		},
		chains.WithModel(*flagLLMModel),
		chains.WithTemperature(0.1),
		chains.WithSeed(123),
		chains.WithCallback(langChainTracer),
	)
	if err != nil {
		return err
	}

	response, ok := out["text"].(string)
	if !ok {
		return fmt.Errorf("invalid response type: %T", out["text"])
	}

	var output Response
	if err := json.Unmarshal([]byte(response), &output); err != nil {
		return fmt.Errorf("unmarshal output: %w", err)
	}

	fmt.Println("")
	fmt.Println("Chain of thought: ")
	fmt.Println(output.ChainOfThought)
	fmt.Println("> Answer: ", output.Answer)
	fmt.Println("> Confidence score: ", output.Confidence)
	return nil
}

func runOutputViaTool(ctx context.Context, llmModel llms.Model, tools []*tools.NativeTool, langsmithClient *langsmith.Client, langchainProject string) error {
	prompTemplates := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("You are a weather and stock expert. store the temperature and stock price", nil),
		prompts.NewHumanMessagePromptTemplate("What is the current weather in {{.location}} and the stock price of {{.symbol}}", nil),
	})

	llmChain := chains.NewLLMChainV2(llmModel, prompTemplates)
	llmChain.RegisterTools(tools...)

	// ----------------------------------------------------------------------------
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
			"symbol":   "AAPL",
		},
		chains.WithModel(*flagLLMModel),
		chains.WithTemperature(0.1),
		chains.WithSeed(123),
		chains.WithCallback(langChainTracer),
	)
	if err != nil {
		return err
	}

	response, ok := out["text"].(string)
	if !ok {
		return fmt.Errorf("invalid response type: %T", out["text"])
	}
	fmt.Print(response)
	return nil
}

func schemaPrompt() string {
	resp := &Response{
		ChainOfThought: "The current weather in Montreal, QC is 20 degrees Celsius with a chance of rain.",
		Answer:         "The current weather in Montreal, QC is 20 degrees Celsius with a chance of rain.",
		Confidence:     0.9,
	}

	schema, err := getJsonSchema(resp)
	if err != nil {
		panic(err)
	}

	schemaStr, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		panic(err)
	}

	exampleStr, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`
	Answer in a JSON format that respects the following JSON schema.
	When the JSON schema specifies an enum list for a string, ensure that the returned value is actually in the list of allowed values, defined of the "enum" field and mention the available value in the 'chain_of_thought' field of your response.
	
	%s
	
	Here is an example of a valid JSON output
	
	%s
`, schemaStr, exampleStr)
}

type Response struct {
	ChainOfThought string  `json:"chain-of-thought" jsonschema_description:"Explanation of the chain of thought leading to the question being answered"`
	Answer         string  `json:"answer" jsonschema_description:"answer the question"`
	Confidence     float64 `json:"confidence-score" jsonschema_description:"Confidence score of the answer. It should be between 0 and 1"`
}

func getJsonSchema(in any) (map[string]any, error) {
	r := jsonschema.Reflector{}
	r.AssignAnchor = false
	r.Anonymous = true
	r.AllowAdditionalProperties = false
	r.DoNotReference = true
	schema := r.ReflectFromType(reflect.TypeOf(in))

	cnt, err := schema.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json schema: %w", err)
	}
	out := map[string]any{}

	if err := json.Unmarshal(cnt, &out); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json schema: %w", err)
	}
	delete(out, "$schema")
	delete(out, "additionalProperties")
	return out, nil
}
