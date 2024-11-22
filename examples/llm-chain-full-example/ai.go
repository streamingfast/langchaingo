package main

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getLLMModel() (llms.Model, error) {
	if openAPIKey := os.Getenv("OPENAI_API_KEY"); openAPIKey != "" {
		organization := os.Getenv("OPENAI_ORGANIZATION")
		if organization == "" {
			return nil, fmt.Errorf("OPENAI_ORGANIZATION is required")
		}
		return openai.New(
			openai.WithToken(openAPIKey),
			openai.WithOrganization(organization),
		)
	}

	if anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicAPIKey != "" {
		return anthropic.New(anthropic.WithToken(anthropicAPIKey))
	}

	return nil, fmt.Errorf("no llm model found")
}

func getPromptTemplate() prompts.ChatPromptTemplate {
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
	chatPrompts := []prompts.MessageFormatter{
		prompts.NewSystemMessagePromptTemplate("You are a translation expert", nil),
		prompts.NewSystemMessagePromptTemplate(schemaPrompt, nil),
		prompts.NewHumanMessagePromptTemplate("What is the current weather in {{.location}}?", nil),
	}

	return prompts.NewChatPromptTemplate(chatPrompts)
}

type LangchainLogger struct {
	*zap.Logger
}

func (l *LangchainLogger) Debugf(format string, v ...interface{}) {
	l.log(zap.DebugLevel, format, v...)
}

func (l *LangchainLogger) Errorf(format string, v ...interface{}) {
	l.log(zap.ErrorLevel, format, v...)
}

func (l *LangchainLogger) Infof(format string, v ...interface{}) {
	l.log(zap.InfoLevel, format, v...)
}

func (l *LangchainLogger) Warnf(format string, v ...interface{}) {
	l.log(zap.WarnLevel, format, v...)
}

func (l *LangchainLogger) log(Level zapcore.Level, format string, v ...interface{}) {
	if ce := l.Check(Level, fmt.Sprintf(format, v...)); ce != nil {
		ce.Write()
	}
}
