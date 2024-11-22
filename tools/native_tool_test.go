package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type test[I any, O any] struct {
	name string
	cb   NativeToolCallFunc[I, O]
}

func Test_NewNativeTool(t *testing.T) {
	out, err := NewNativeTool(getCurrentWeather, "Get Current Weather Description")
	require.NoError(t, err)

	assert.Equal(t, "getCurrentWeather", out.name)
	assert.Equal(t, "Get Current Weather Description", out.description)
	assert.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]interface{}{
			"location": map[string]interface{}{
				"type":        "string",
				"description": "The city and state, e.g. San Francisco, CA",
			},
			"unit": map[string]any{
				"type": "string",
				"enum": []interface{}{"fahrenheit", "celsius"},
			},
		},
		"required": []interface{}{"location", "unit"},
	}, out.jsonSchema)

	out2, err := NewNativeTool(getStockPrice, "Get Current stock price")
	require.NoError(t, err)

	assert.Equal(t, "getStockPrice", out2.name)
	assert.Equal(t, "Get Current stock price", out2.description)
	assert.Equal(t, map[string]any{
		"type": "object",
		"properties": map[string]interface{}{
			"symbol": map[string]interface{}{
				"type":        "string",
				"description": "The stock symbol",
			},
		},
		"required": []interface{}{"symbol"},
	}, out2.jsonSchema)
}

func Test_getTollCallFunction(t *testing.T) {
	input := `{"location": "San Francisco, CA","unit": "celsius"}`
	callback := getNativeToolCallFunction[*WeatherInput, *WeatherOutput](reflect.TypeOf(&WeatherInput{}), getCurrentWeather)
	output, err := callback(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, `{"location":"San Francisco, CA","unit":"fahrenheit","temp":"72"}`, output)

	input = `{"symbol": "GOOG"}`
	callback = getNativeToolCallFunction[StockPriceInput, StockPriceOut](reflect.TypeOf(StockPriceInput{}), getStockPrice)
	output, err = callback(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, `{"price":{"spot":1,"future":2}}`, output)

}

type WeatherInput struct {
	Location string `json:"location" jsonschema_description:"The city and state, e.g. San Francisco, CA"`
	Unit     string `json:"unit" jsonschema:"enum=fahrenheit,enum=celsius"`
}

type WeatherOutput struct {
	Location string `json:"location"`
	Unit     string `json:"unit"`
	Temp     string `json:"temp"`
}

func getCurrentWeather(ctx context.Context, in *WeatherInput) (*WeatherOutput, error) {
	cnt, _ := json.Marshal(in)
	fmt.Println("getCurrentWeather: ", string(cnt))
	return &WeatherOutput{
		Location: "San Francisco, CA",
		Unit:     "fahrenheit",
		Temp:     "72",
	}, nil
}

type StockPriceInput struct {
	Symbol string `json:"symbol" jsonschema_description:"The stock symbol"`
}

type StockPriceOut struct {
	Price struct {
		Spot   float64 `json:"spot"`
		Future float64 `json:"future"`
	} `json:"price"`
}

func getStockPrice(ctx context.Context, in StockPriceInput) (StockPriceOut, error) {
	cnt, _ := json.Marshal(in)
	fmt.Println("getStockPrice: ", string(cnt))
	return StockPriceOut{
		Price: struct {
			Spot   float64 `json:"spot"`
			Future float64 `json:"future"`
		}{
			Spot:   1,
			Future: 2,
		},
	}, nil
}
