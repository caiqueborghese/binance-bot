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

// EvaluateSignal aplica uma estratégia combinando MACD, RSI e Volume MA
func EvaluateSignal(klines [][]interface{}) Signal {
	closes := indicators.ExtractClosePrices(klines)
	volumes := indicators.ExtractVolumes(klines)

	// Parâmetros fixos
	fastPeriod := 12
	slowPeriod := 26
	signalPeriod := 9
	rsiPeriod := 14
	volumeMAPeriod := 20

	// Calcula os indicadores
	macd, _, _ := indicators.ComputeMACD(closes, fastPeriod, slowPeriod, signalPeriod)
	rsi := indicators.ComputeRSI(closes, rsiPeriod)
	volMA := indicators.ComputeVolumeMA(volumes, volumeMAPeriod)

	last := len(closes) - 1
	if last < slowPeriod || last < rsiPeriod || last < volumeMAPeriod {
		return NoSignal
	}

	// Critérios para COMPRA
	if macd[last] > 0 && rsi[last] > 50 && volumes[last] > volMA[last] {
		return BuySignal
	}

	// Critérios para VENDA
	if macd[last] < 0 && rsi[last] < 50 && volumes[last] > volMA[last] {
		return SellSignal
	}

	return NoSignal
}
