package chains

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/tmc/langchaingo/callbacks"
	"github.com/tmc/langchaingo/schema"
)

// Key name used to store the intermediate steps in the output, when configured.
const _intermediateStepsOutputKey = "intermediateSteps"

// Chain is the interface all chains must implement.
type Chain interface {
	// Call runs the logic of the chain and returns the output. This method should
	// not be called directly. Use rather the Call, Run or Predict functions that
	// handles the memory and other aspects of the chain.
	Call(ctx context.Context, inputs map[string]any, options ...ChainCallOption) (map[string]any, error)
	// GetMemory gets the memory of the chain.
	GetMemory() schema.Memory
	// GetInputKeys returns the input keys the chain expects.
	GetInputKeys() []string
	// GetOutputKeys returns the output keys the chain returns.
	GetOutputKeys() []string
}

// Call is the standard function used for executing chains.
func Call(ctx context.Context, c Chain, inputValues map[string]any, options ...ChainCallOption) (map[string]any, error) { // nolint: lll
	ctx = setupChainCallbackHandler(ctx, c, options)

	fullValues := make(map[string]any, 0)
	for key, value := range inputValues {
		fullValues[key] = value
	}

	newValues, err := c.GetMemory().LoadMemoryVariables(ctx, inputValues)
	if err != nil {
		return nil, err
	}

	for key, value := range newValues {
		fullValues[key] = value
	}

	chainCallbackHandlers := callbacks.CallbackHandler(ctx)

	if chainCallbackHandlers != nil {
		chainCallbackHandlers.HandleChainStart(ctx, inputValues)
	}

	outputValues, err := callChain(ctx, c, fullValues, options...)
	if err != nil {
		if chainCallbackHandlers != nil {
			chainCallbackHandlers.HandleChainError(ctx, err)
		}
		return outputValues, err
	}

	if chainCallbackHandlers != nil {
		chainCallbackHandlers.HandleChainEnd(ctx, outputValues) // Call the chain end
	}

	if err = c.GetMemory().SaveContext(ctx, inputValues, outputValues); err != nil {
		return outputValues, err
	}

	return outputValues, nil
}

// CallInto calls the chain and unmarshals the output in the value pointed to by v. If v is nil or not a pointer, returns an error
func CallInto(ctx context.Context, c Chain, inputValues map[string]any, v any, options ...ChainCallOption) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("v is not a pointer")
	}

	outputValues, err := Call(ctx, c, inputValues, options...)
	if err != nil {
		return err
	}

	outputKeys := c.GetOutputKeys()
	if len(outputKeys) == 0 {
		return fmt.Errorf("expected at least one output key, got %d", len(outputKeys))
	}

	outputString, ok := outputValues[outputKeys[0]].(string)
	if !ok {
		return fmt.Errorf("expected output key %s to be a string, got %T", outputKeys[0], outputValues[outputKeys[0]])
	}

	return json.Unmarshal([]byte(outputString), v)
}

// setupChainCallbackHandler sets up the chain callback handler in the context, which will be passed down to Chain calls.
// we will prioritize a callback handler set in the options, and fallback to a chain specific if set.
func setupChainCallbackHandler(ctx context.Context, c Chain, options []ChainCallOption) context.Context {
	// if callback handler is set in options, prioritize that
	if handler := getChainCallCallbackHandler(options); handler != nil {
		return callbacks.WithCallback(ctx, handler)
	}

	if handler := getChainCallbackHandler(c); handler != nil {
		return callbacks.WithCallback(ctx, handler)
	}
	return ctx
}

func callChain(
	ctx context.Context,
	c Chain,
	fullValues map[string]any,
	options ...ChainCallOption,
) (map[string]any, error) {
	if err := validateInputs(c, fullValues); err != nil {
		return nil, err
	}

	outputValues, err := c.Call(ctx, fullValues, options...)
	if err != nil {
		return outputValues, err
	}
	if err := validateOutputs(c, outputValues); err != nil {
		return outputValues, err
	}

	return outputValues, nil
}

// Run can be used to execute a chain if the chain only expects one input and
// one string output.
func Run(ctx context.Context, c Chain, input any, options ...ChainCallOption) (string, error) {
	inputKeys := c.GetInputKeys()
	memoryKeys := c.GetMemory().MemoryVariables(ctx)
	neededKeys := make([]string, 0, len(inputKeys))

	// Remove keys gotten from the memory.
	for _, inputKey := range inputKeys {
		isInMemory := false
		for _, memoryKey := range memoryKeys {
			if inputKey == memoryKey {
				isInMemory = true
				continue
			}
		}
		if isInMemory {
			continue
		}
		neededKeys = append(neededKeys, inputKey)
	}
	if len(neededKeys) != 1 {
		return "", ErrMultipleInputsInRun
	}

	outputKeys := c.GetOutputKeys()
	if len(outputKeys) != 1 {
		return "", ErrMultipleOutputsInRun
	}

	inputValues := map[string]any{neededKeys[0]: input}
	outputValues, err := Call(ctx, c, inputValues, options...)
	if err != nil {
		return "", err
	}

	outputValue, ok := outputValues[outputKeys[0]].(string)
	if !ok {
		return "", ErrWrongOutputTypeInRun
	}

	return outputValue, nil
}

// Predict can be used to execute a chain if the chain only expects one string output.
func Predict(ctx context.Context, c Chain, inputValues map[string]any, options ...ChainCallOption) (string, error) {
	outputValues, err := Call(ctx, c, inputValues, options...)
	if err != nil {
		return "", err
	}

	outputKeys := c.GetOutputKeys()
	if len(outputKeys) != 1 {
		return "", ErrMultipleOutputsInPredict
	}

	outputValue, ok := outputValues[outputKeys[0]].(string)
	if !ok {
		return "", ErrOutputNotStringInPredict
	}

	return outputValue, nil
}

const _defaultApplyMaxNumberWorkers = 5

type applyInputJob struct {
	input map[string]any
	i     int
}

type applyResult struct {
	result map[string]any
	err    error
	i      int
}

// Apply executes the chain for each of the inputs asynchronously.
func Apply(ctx context.Context, c Chain, inputValues []map[string]any, maxWorkers int, options ...ChainCallOption) ([]map[string]any, error) { // nolint:lll
	if maxWorkers <= 0 {
		maxWorkers = _defaultApplyMaxNumberWorkers
	}

	inputJobs := make(chan applyInputJob, len(inputValues))
	resultsChan := make(chan applyResult, len(inputValues))

	var wg sync.WaitGroup
	wg.Add(maxWorkers)

	for w := 0; w < maxWorkers; w++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case input, ok := <-inputJobs:
					if !ok {
						return
					}
					res, err := Call(ctx, c, input.input, options...)
					resultsChan <- applyResult{
						result: res,
						err:    err,
						i:      input.i,
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	sendApplyInputJobs(inputJobs, inputValues)
	return getApplyResults(ctx, resultsChan, inputValues)
}

func sendApplyInputJobs(inputJobs chan applyInputJob, inputValues []map[string]any) {
	for i, input := range inputValues {
		inputJobs <- applyInputJob{
			input: input,
			i:     i,
		}
	}
	close(inputJobs)
}

func getApplyResults(ctx context.Context, resultsChan chan applyResult, inputValues []map[string]any) ([]map[string]any, error) { //nolint:lll
	results := make([]map[string]any, len(inputValues))
	for range results {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-resultsChan:
			if r.err != nil {
				return nil, r.err
			}
			results[r.i] = r.result
		}
	}

	return results, nil
}

func validateInputs(c Chain, inputValues map[string]any) error {
	for _, k := range c.GetInputKeys() {
		if _, ok := inputValues[k]; !ok {
			return fmt.Errorf("%w: %w: %v", ErrInvalidInputValues, ErrMissingInputValues, k)
		}
	}
	return nil
}

func validateOutputs(c Chain, outputValues map[string]any) error {
	for _, k := range c.GetOutputKeys() {
		if _, ok := outputValues[k]; !ok {
			return fmt.Errorf("%w: %v", ErrInvalidOutputValues, k)
		}
	}
	return nil
}

func getChainCallbackHandler(c Chain) callbacks.Handler {
	if handlerHaver, ok := c.(callbacks.HandlerHaver); ok {
		return handlerHaver.GetCallbackHandler()
	}
	return nil
}
