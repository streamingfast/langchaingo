package main

import (
	"context"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema_description:"The city and state, e.g. San Francisco, CA"`
	Unit     string `json:"unit" jsonschema:"enum=fahrenheit,enum=celsius"`
}

type WeatherOutput struct {
	Location string `json:"location"`
	Unit     string `json:"unit"`
	Temp     string `json:"temp"`
}

func getCurrentWeather(_ context.Context, in *WeatherInput) (*WeatherOutput, error) {
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

func getStockPrice(_ context.Context, _ StockPriceInput) (StockPriceOut, error) {
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
