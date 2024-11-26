package tracing

type Traceable interface {
	SetLLMMetadata(metadata *TracerLLMMetadata)
}

type TracerLLMMetadata struct {
	SpanName  string
	ModelName string
	ModelType string
	Provider  string
}
