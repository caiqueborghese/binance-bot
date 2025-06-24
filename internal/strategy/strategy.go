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

// EvaluateSignal aplica uma estratégia mais segura
func EvaluateSignal(klines [][]interface{}) Signal {
	closes := indicators.ExtractClosePrices(klines)
	opens := indicators.ExtractOpenPrices(klines)
	volumes := indicators.ExtractVolumes(klines)

	fastPeriod := 12
	slowPeriod := 26
	signalPeriod := 9
	rsiPeriod := 14
	volPeriod := 20

	macd, signalLine, _ := indicators.ComputeMACD(closes, fastPeriod, slowPeriod, signalPeriod)
	rsi := indicators.ComputeRSI(closes, rsiPeriod)
	volMA := indicators.ComputeVolumeMA(volumes, volPeriod)

	last := len(closes) - 1
	if last < slowPeriod || last < rsiPeriod || last < volPeriod {
		return NoSignal
	}

	// Filtro: vela não pode ser muito maior que a média das últimas 10
	if !indicators.IsCandleReasonable(klines, 10, 1.8) {
		return NoSignal
	}

	// COMPRA: MACD cruzando acima do sinal, RSI > 55, candle de alta, volume alto
	if macd[last] > signalLine[last] && rsi[last] > 55 && closes[last] > opens[last] && volumes[last] > volMA[last] {
		return BuySignal
	}

	// VENDA: MACD cruzando abaixo do sinal, RSI < 45, candle de baixa, volume alto
	if macd[last] < signalLine[last] && rsi[last] < 45 && closes[last] < opens[last] && volumes[last] > volMA[last] {
		return SellSignal
	}

	return NoSignal
}
