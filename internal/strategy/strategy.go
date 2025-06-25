package strategy

import (
	"math"
	"time"

	"binance-bot/internal/indicators"
	"binance-bot/internal/types"
)

const (
	BuySignal  = "BUY"
	SellSignal = "SELL"
	NoSignal   = ""
)

var cooldownMap = make(map[string]int64)
var cooldownDuration = int64(10 * 60) // 10 minutos em segundos

type Candle struct {
	Open  float64
	Close float64
	High  float64
	Low   float64
}

func IsStrongCandle(c Candle) bool {
	rangeSize := c.High - c.Low
	if rangeSize == 0 {
		return false
	}
	bodySize := math.Abs(c.Close - c.Open)
	return bodySize/rangeSize >= 0.6
}

func IsInCooldown(symbol string) bool {
	now := time.Now().Unix()
	last := cooldownMap[symbol]
	return now-last < cooldownDuration
}

func RegisterCooldown(symbol string) {
	cooldownMap[symbol] = time.Now().Unix()
}

func EvaluateSignal(klines []types.Kline, symbol string) string {
	if len(klines) < 35 || IsInCooldown(symbol) {
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
	volumeOK := volumeAtual > volMA

	atr := indicators.ComputeATR(klines, 14)
	atrHist := 0.0
	for i := len(klines) - 15; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		closePrev := klines[i-1].Close
		tr := math.Max(high-low, math.Max(math.Abs(high-closePrev), math.Abs(low-closePrev)))
		atrHist += tr
	}
	atrMedia := atrHist / 14
	atrOK := atr > atrMedia

	last := klines[len(klines)-1]
	candle := Candle{
		Open:  last.Open,
		Close: last.Close,
		High:  last.High,
		Low:   last.Low,
	}
	candleOK := IsStrongCandle(candle)

	cruzamentoAlta := macd > signal && macdPrev < signalPrev
	cruzamentoBaixa := macd < signal && macdPrev > signalPrev

	if cruzamentoAlta && rsi > 50 && rsi < 70 && candleOK && volumeOK && macd > 0 && signal > 0 && atrOK {
		RegisterCooldown(symbol)
		return BuySignal
	}
	if cruzamentoBaixa && rsi < 45 && rsi > 30 && candleOK && volumeOK && macd < 0 && signal < 0 && atrOK {
		RegisterCooldown(symbol)
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
