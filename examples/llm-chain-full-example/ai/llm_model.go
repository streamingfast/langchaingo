package ai

import "fmt"

//go:generate go-enum -f=$GOFILE

// ENUM(gpt-4o,gpt-4o-mini,claude-3-5-haiku,claude-3-5-sonnet).
type LLMModel string

func NewLLMModel(in string) (LLMModel, error) {
	llmModel := LLMModel(in)
	if !llmModel.IsValid() {
		return "", fmt.Errorf("invalid llm model: %s", in)
	}

	return llmModel, nil
}
