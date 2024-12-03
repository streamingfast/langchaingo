package main

import (
	"fmt"
	"os"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/openai"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func getLLMModel() (llms.Model, error) {
	if openAPIKey := os.Getenv("OPENAI_API_KEY"); openAPIKey != "" {
		organization := os.Getenv("OPENAI_ORGANIZATION")
		if organization == "" {
			return nil, fmt.Errorf("OPENAI_ORGANIZATION is required")
		}
		logger.Info("Using OpenAI model")
		return openai.New(
			openai.WithToken(openAPIKey),
			openai.WithOrganization(organization),
		)
	}

	if anthropicAPIKey := os.Getenv("ANTHROPIC_API_KEY"); anthropicAPIKey != "" {
		logger.Info("Using Anthropic model")
		return anthropic.New(anthropic.WithToken(anthropicAPIKey))
	}

	return nil, fmt.Errorf("no llm model found")
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
