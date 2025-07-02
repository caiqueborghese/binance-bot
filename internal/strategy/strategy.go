package strategy

import (
	"binance-bot/internal/indicators"
	"binance-bot/internal/types"
)

type Signal int

const (
	NoSignal Signal = iota
	BuySignal
	SellSignal
)

func EvaluateSignal(klines []types.Kline, symbol string) Signal {
	closes := indicators.ExtractClosePrices(klines)
	volumes := indicators.ExtractVolumes(klines)
	macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
	rsi := indicators.ComputeRSI(closes, 14)
	volumeMA := indicators.ComputeVolumeMA(volumes, 20)

	if len(closes) < 6 || len(macdLine) < 2 || len(signalLine) < 2 || len(rsi) < 1 {
		return NoSignal
	}

	latestClose := closes[len(closes)-1]
	latestVolume := volumes[len(volumes)-1]
	latestRSI := rsi[len(rsi)-1]
	macdPrev := macdLine[len(macdLine)-2]
	macdCurr := macdLine[len(macdLine)-1]
	signalPrev := signalLine[len(signalLine)-2]
	signalCurr := signalLine[len(signalLine)-1]

	high5 := klines[len(klines)-6].High
	low5 := klines[len(klines)-6].Low
	for i := len(klines) - 6; i < len(klines)-1; i++ {
		if klines[i].High > high5 {
			high5 = klines[i].High
		}
		if klines[i].Low < low5 {
			low5 = klines[i].Low
		}
	}

	// BUY condition
	if latestRSI > 60 &&
		macdPrev < signalPrev && macdCurr > signalCurr &&
		latestVolume > 1.5*volumeMA &&
		latestClose > high5 {
		return BuySignal
	}

	// SELL condition
	if latestRSI < 40 &&
		macdPrev > signalPrev && macdCurr < signalCurr &&
		latestVolume > 1.5*volumeMA &&
		latestClose < low5 {
		return SellSignal
	}

	return NoSignal
}
