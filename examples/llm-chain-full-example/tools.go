package main

import (
	"context"

	"go.uber.org/zap"
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

type Tools struct {
	logger *zap.Logger
}

func (t *Tools) getCurrentWeather(_ context.Context, in *WeatherInput) (*WeatherOutput, error) {
	t.logger.Info("getCurrentWeather", zap.Reflect("input", in))
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

func (t *Tools) getStockPrice(_ context.Context, in StockPriceInput) (StockPriceOut, error) {
	t.logger.Info("getStockPrice", zap.Reflect("input", in))
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

type RecordInput struct {
	Temp  string `json:"temp" jsonschema_description:"The temperature in human readable format"`
	Price string `json:"price" jsonschema_description:"The price in human readable format"`
}

func (t *Tools) storeRecord(_ context.Context, out RecordInput) (string, error) {
	t.logger.Info("storeRecord", zap.Reflect("input", out))
	return "ok", nil
}
