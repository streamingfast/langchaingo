package langsmith

type LangChainTracerOption interface {
	apply(t *LangChainTracer)
}

type langChainTracerOptionFunc func(t *LangChainTracer)

func (f langChainTracerOptionFunc) apply(t *LangChainTracer) {
	f(t)
}

func WithClient(client *Client) LangChainTracerOption {
	return langChainTracerOptionFunc(func(t *LangChainTracer) {
		t.client = client
	})
}

func WithName(name string) LangChainTracerOption {
	return langChainTracerOptionFunc(func(t *LangChainTracer) {
		t.name = name
	})
}

func WithProjectName(projectName string) LangChainTracerOption {
	return langChainTracerOptionFunc(func(t *LangChainTracer) {
		t.projectName = projectName
	})
}

func WithExampleID(exampleID string) LangChainTracerOption {
	return langChainTracerOptionFunc(func(t *LangChainTracer) {
		t.exampleID = exampleID
	})
}

func WithApiKey(apiKey string) LangChainTracerOption {
	return langChainTracerOptionFunc(func(t *LangChainTracer) {
		t.client.apiKey = apiKey
	})
}
