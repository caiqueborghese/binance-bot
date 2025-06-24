package strategy

import (
	"math"

	"binance-bot/internal/indicators"
	"binance-bot/internal/types"
)

const (
	BuySignal  = "BUY"
	SellSignal = "SELL"
	NoSignal   = ""
)

// Candle representa uma vela individual
type Candle struct {
	Open  float64
	Close float64
	High  float64
	Low   float64
}

// IsStrongCandle avalia se o corpo da vela representa força de direção (>60% do range total)
func IsStrongCandle(c Candle) bool {
	rangeSize := c.High - c.Low
	if rangeSize == 0 {
		return false
	}
	bodySize := math.Abs(c.Close - c.Open)
	return bodySize/rangeSize >= 0.6
}

// EvaluateSignal aplica a estratégia com múltiplos filtros
func EvaluateSignal(klines []types.Kline) string {
	if len(klines) < 35 {
		return NoSignal
	}

	closes := indicators.ExtractClosePrices(klines)
	volumes := indicators.ExtractVolumes(klines)

	// MACD
	macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
	macd := macdLine[len(macdLine)-1]
	macdPrev := macdLine[len(macdLine)-2]
	signal := signalLine[len(signalLine)-1]
	signalPrev := signalLine[len(signalLine)-2]

	// RSI
	rsiValues := indicators.ComputeRSI(closes, 14)
	rsi := rsiValues[len(rsiValues)-1]

	// Volume
	volMA := indicators.ComputeVolumeMA(volumes, 14)
	volumeAtual := volumes[len(volumes)-1]
	volumeOK := volumeAtual > volMA

	// Última vela
	last := klines[len(klines)-1]
	candle := Candle{
		Open:  last.Open,
		Close: last.Close,
		High:  last.High,
		Low:   last.Low,
	}
	candleOK := IsStrongCandle(candle)

	// Cruzamentos MACD
	cruzamentoAlta := macd > signal && macdPrev < signalPrev
	cruzamentoBaixa := macd < signal && macdPrev > signalPrev

	if cruzamentoAlta && rsi > 50 && candleOK && volumeOK {
		return BuySignal
	}
	if cruzamentoBaixa && rsi < 50 && candleOK && volumeOK {
		return SellSignal
	}
	return NoSignal
}
