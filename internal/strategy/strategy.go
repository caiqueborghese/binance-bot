package strategy

import (
	"binance-bot/internal/indicators"
	"binance-bot/internal/types"
)

const (
	BuySignal  = "BUY"
	SellSignal = "SELL"
	NoSignal   = ""
)

type Candle struct {
	Open  float64
	Close float64
	High  float64
	Low   float64
}

// Estratégia agressiva: múltiplas entradas, menos filtros, foco em ganho
func EvaluateSignal(klines []types.Kline, symbol string) string {
	if len(klines) < 35 {
		return NoSignal
	}

	closes := indicators.ExtractClosePrices(klines)
	volumes := indicators.ExtractVolumes(klines)

	macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
	macd := macdLine[len(macdLine)-1]
	macdPrev := macdLine[len(macdLine)-2]
	signal := signalLine[len(signalLine)-1]
	signalPrev := signalLine[len(signalLine)-2]

	rsiValues := indicators.ComputeRSI(closes, 14)
	rsi := rsiValues[len(rsiValues)-1]

	volMA := indicators.ComputeVolumeMA(volumes, 14)
	volumeAtual := volumes[len(volumes)-1]
	volumeOK := volumeAtual >= volMA*0.8

	cruzamentoAlta := macd > signal && macdPrev < signalPrev
	cruzamentoBaixa := macd < signal && macdPrev > signalPrev

	if cruzamentoAlta && rsi > 50 && volumeOK {
		return BuySignal
	}
	if cruzamentoBaixa && rsi < 50 && volumeOK {
		return SellSignal
	}
	return NoSignal
}

func ComputeTrailingStops(entry float64, side string, atr float64) (takeProfit, stopLoss float64) {
	multTP := 2.5
	multSL := 1.5
	if side == BuySignal {
		takeProfit = entry + (atr * multTP)
		stopLoss = entry - (atr * multSL)
	} else {
		takeProfit = entry - (atr * multTP)
		stopLoss = entry + (atr * multSL)
	}
	return
}
