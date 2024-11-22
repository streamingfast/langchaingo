package ai

import (
	"context"
	"fmt"
	"text/template"

	"github.com/tmc/langchaingo/prompts"
	"go.uber.org/zap"
)

type Variable map[string]any

func (v Variable) WithVariable(key string, value any) Variable {
	v[key] = value
	return v
}

type Prompt struct {
	Model      LLMModel
	PromptTmpl string
	SchemaTmpl string
	HumanTmpl  string
	Vars       Variable
}

func (c *Client) RunPrompt(ctx context.Context, debugPrefix string, prompt *Prompt, runID string, logger *zap.Logger) ([]byte, error) {

	p, debugTemplates := prompt.getPromptTemplate(debugPrefix)

	c.debugTemplates(ctx, runID, prompt.Vars, debugTemplates, logger)

	output, err := c.call(ctx, runID, p, prompt.Vars, prompt.Model, logger)
	if err != nil {
		return nil, fmt.Errorf("llm: %w", err)
	}

	c.saveOutput(ctx, runID, fmt.Sprintf("%s.output", debugPrefix), []byte(output), logger)

	return []byte(output), nil
}

func (p *Prompt) getPromptTemplate(templatePrefix string) (prompts.ChatPromptTemplate, []*template.Template) {
	var chatPrompts []prompts.MessageFormatter
	var debugTemplates []*template.Template
	if p.PromptTmpl != "" {
		chatPrompts = append(chatPrompts, prompts.NewSystemMessagePromptTemplate(p.PromptTmpl, nil))
		debugTemplates = append(debugTemplates, template.Must(template.New(fmt.Sprintf("%s.prompt.gotmpl", templatePrefix)).Parse(p.PromptTmpl)))
	}
	if p.SchemaTmpl != "" {
		chatPrompts = append(chatPrompts, prompts.NewSystemMessagePromptTemplate(p.SchemaTmpl, nil))
		debugTemplates = append(debugTemplates, template.Must(template.New(fmt.Sprintf("%s.schema.gotmpl", templatePrefix)).Parse(p.SchemaTmpl)))
	}
	if p.HumanTmpl != "" {
		chatPrompts = append(chatPrompts, prompts.NewHumanMessagePromptTemplate(p.HumanTmpl, nil))
		debugTemplates = append(debugTemplates, template.Must(template.New(fmt.Sprintf("%s.human.gotmpl", templatePrefix)).Parse(p.HumanTmpl)))

	}
	return prompts.NewChatPromptTemplate(chatPrompts), debugTemplates
}
