package strategy

import (
	"binance-bot/internal/indicators"
	"binance-bot/internal/types"
)

// Sinais possíveis
const (
	NoSignal = iota
	BuySignal
	SellSignal
)

// EvaluateSignal aplica uma estratégia agressiva e altamente assertiva
func EvaluateSignal(klines []types.Kline, symbol string) int {
	if len(klines) < 35 {
		return NoSignal
	}

	closes := indicators.ExtractClosePrices(klines)
	volumes := indicators.ExtractVolumes(klines)

	rsi := indicators.ComputeRSI(closes, 14)
	macd, signal, hist := indicators.ComputeMACD(closes, 12, 26, 9)
	volMA := indicators.ComputeVolumeMA(volumes, 10)

	if len(rsi) >= 2 && len(hist) >= 3 && len(macd) >= 2 && len(signal) >= 2 {
		rsi1 := rsi[len(rsi)-2]
		rsi2 := rsi[len(rsi)-1]
		hist1 := hist[len(hist)-3]
		hist2 := hist[len(hist)-2]
		hist3 := hist[len(hist)-1]
		vol := volumes[len(volumes)-1]

		// BUY
		if rsi1 < 50 && rsi2 > 50 && hist1 < hist2 && hist2 < hist3 && vol > volMA {
			return BuySignal
		}

		// SELL
		if rsi1 > 50 && rsi2 < 50 && hist1 > hist2 && hist2 > hist3 && vol > volMA {
			return SellSignal
		}
	}

	return NoSignal
}
