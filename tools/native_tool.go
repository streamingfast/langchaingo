package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/invopop/jsonschema"
)

//nolint:gochecknoglobals
var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()

func NewNativeTool[I any, O any](toolCall NativeToolCallFunc[I, O], description string) (*NativeTool, error) {
	funcName := getFunctionName(toolCall)
	toolCallType := reflect.TypeOf(toolCall)

	// Get number of input parameters
	numIn := toolCallType.NumIn()
	if numIn != 2 {
		return nil, fmt.Errorf("toolCall must have 2 input parameters")
	}

	firstArg := toolCallType.In(0)
	if firstArg.Kind() == reflect.Pointer {
		return nil, fmt.Errorf("first argument must not be a pointer")
	}

	if firstArg != contextType {
		return nil, fmt.Errorf("first argument must be context.Context")
	}

	secondArg := toolCallType.In(1)

	var structPtr reflect.Value

	var properties map[string]any
	var err error
	if secondArg.Kind() == reflect.Pointer {
		// setups up a pointer pf the struct
		structConcreteType := secondArg.Elem()
		structPtr = reflect.New(structConcreteType)
		structPtr.Elem().Set(reflect.Zero(structConcreteType))
		properties, err = getJSONSchema(structPtr.Interface().(I))
	} else {
		structConcreteType := secondArg
		structPtr = reflect.New(structConcreteType)
		structPtr.Elem().Set(reflect.Zero(structConcreteType))
		properties, err = getJSONSchema(structPtr.Interface().(*I))
	}
	if err != nil {
		return nil, fmt.Errorf("get json schema: %w", err)
	}

	return &NativeTool{
		name:        funcName,
		description: description,
		call:        getNativeToolCallFunction(secondArg, toolCall),
		jsonSchema:  properties,
	}, nil
}

func getNativeToolCallFunction[I any, O any](toolCallInput reflect.Type, toolCallFunc NativeToolCallFunc[I, O]) func(ctx context.Context, input string) (output string, err error) {
	return func(ctx context.Context, input string) (string, error) {
		var funcInput I
		var funcOutput O
		var ok bool

		// nolint:nestif
		if toolCallInput.Kind() == reflect.Pointer {
			// This flow occurs when your toolCallFunc input is a pointer of a struct (e.g *WeatherInput), note I = pointer of struct
			// structConcreteType is the struct type (e.g. WeatherInput)
			structConcreteType := toolCallInput.Elem()
			// we are create a new pointer of the struct (*WeatherInput)
			inputStructPtr := reflect.New(structConcreteType)
			// we are unmarshalling the JSON input into this pointer
			if err := json.Unmarshal([]byte(input), inputStructPtr.Interface()); err != nil {
				return "", fmt.Errorf("unmarshal input: %w", err)
			}
			// since I is a pointer of struct and inputStructPtr is also that we cast
			funcInput, ok = inputStructPtr.Interface().(I)
			if !ok {
				return "", fmt.Errorf("invalid input type")
			}
		} else {
			// This flow occurs when your toolCallFunc input is  a struct (e.g WeatherInput), note I = struct
			// we are create a new pointer of the struct (*WeatherInput)
			inputStructPtr := reflect.New(toolCallInput)
			// we are unmarshalling the JSON input into this pointer
			if err := json.Unmarshal([]byte(input), inputStructPtr.Interface()); err != nil {
				return "", fmt.Errorf("unmarshal input: %w", err)
			}
			// Since I is the struct and inputStructPtr is the pointer of the struct we will get the Element (deference) and cast it
			funcInput, ok = inputStructPtr.Elem().Interface().(I)
			if !ok {
				return "", fmt.Errorf("invalid input type")
			}
		}

		funcOutput, err := toolCallFunc(ctx, funcInput)
		if err != nil {
			return "", fmt.Errorf("tool call failed: %w", err)
		}
		outputJSON, err := json.Marshal(funcOutput)
		if err != nil {
			return "", fmt.Errorf("failed to marshal output: %w", err)
		}
		return string(outputJSON), nil
	}
}

func getJSONSchema(in any) (map[string]any, error) {
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

func getFunctionName[I any, O any](toolCall NativeToolCallFunc[I, O]) string {
	in := runtime.FuncForPC(reflect.ValueOf(toolCall).Pointer()).Name()
	chunks := strings.Split(in, "/")
	fullName := chunks[len(chunks)-1]
	nameChunk := strings.Split(fullName, ".")
	return nameChunk[len(nameChunk)-1]
}
