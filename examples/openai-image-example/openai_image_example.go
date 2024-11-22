package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
)

func main() {
	llm, err := openai.New(openai.WithModel("gpt-4o"))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()

	imageBytes, err := os.ReadFile("./sample-image.png")
	if err != nil {
		log.Fatal(err)
	}

	imageBase := "data:image/png;base64, " + base64.StdEncoding.EncodeToString(imageBytes)

	translatePrompt := prompts.NewChatPromptTemplate([]prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("You are an image analysis expert", nil),
		prompts.NewHumanMessagePromptTemplate("Describe the image", nil),
		// we add a placeholder prompt that will inject the images
		prompts.MessagesPlaceholder{VariableName: "Images"},
	})

	llmChain := chains.NewLLMChain(llm, translatePrompt)
	images := []llms.ChatMessage{
		llms.ImageChatMessage{Content: imageBase},
	}

	outputValues, err := chains.Call(ctx, llmChain, map[string]any{
		// This needs to match the VariableName in the MessagesPlaceholder
		"Images": images,
	})
	if err != nil {
		log.Fatal(err)
	}

	cnt, err := json.Marshal(outputValues)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(cnt))
}
