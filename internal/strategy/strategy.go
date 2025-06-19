// internal/strategy/strategy.go
package strategy

import (
	"binance-bot/internal/indicators"
)

// Signal representa um sinal de trading
type Signal int

const (
	NoSignal Signal = iota
	BuySignal
	SellSignal
)

// EvaluateSignal applies the hyper-scalping grid martingale strategy
// Returns BuySignal, SellSignal or NoSignal for the last candle
func EvaluateSignal(klines [][]interface{}) Signal {
	// Extract prices
	closes := indicators.ExtractClosePrices(klines)

	last := len(closes) - 1

	// Buy if price increased
	if closes[last] > closes[last-1] {
		return BuySignal
	}

	// Sell if price decreased
	if closes[last] < closes[last-1] {
		return SellSignal
	}

	return NoSignal
}
