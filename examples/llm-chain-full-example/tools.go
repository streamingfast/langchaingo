package main

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/tools"
)

var getCurrentWeatherToolCall *tools.NativeTool
var getStockPriceToolCall *tools.NativeTool

func init() {
	var err error
	getCurrentWeatherToolCall, err = tools.NewNativeTool(getCurrentWeather, "Get a location's current weather")
	if err != nil {
		panic(err)
	}

	getStockPriceToolCall, err = tools.NewNativeTool(getStockPrice, "Get a symbol's stock price")
	if err != nil {
		panic(err)
	}
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
	out := &WeatherOutput{
		Location: in.Location,
		Unit:     in.Unit,
		Temp:     "26",
	}
	if in.Unit == "fahrenheit" {
		out.Temp = "72"
	}
	return out, nil
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
	fmt.Println("Getting stock price for", in.Symbol)
	return StockPriceOut{
		Price: struct {
			Spot   float64 `json:"spot"`
			Future float64 `json:"future"`
		}{
			Spot:   169.23,
			Future: 236.12,
		},
	}, nil
}
