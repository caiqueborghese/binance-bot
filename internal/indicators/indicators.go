package indicators

import (
	"binance-bot/internal/types"
	"fmt"
	"math"
	"strconv"
)

// Conversor de [][]interface{} para []types.Kline
func ConvertToKlines(data [][]interface{}) []types.Kline {
	var klines []types.Kline
	for _, k := range data {
		openTime, _ := k[0].(float64)
		open, _ := strconv.ParseFloat(fmt.Sprintf("%v", k[1]), 64)
		high, _ := strconv.ParseFloat(fmt.Sprintf("%v", k[2]), 64)
		low, _ := strconv.ParseFloat(fmt.Sprintf("%v", k[3]), 64)
		closePrice, _ := strconv.ParseFloat(fmt.Sprintf("%v", k[4]), 64)
		volume, _ := strconv.ParseFloat(fmt.Sprintf("%v", k[5]), 64)
		closeTime, _ := k[6].(float64)

		klines = append(klines, types.Kline{
			OpenTime:  int64(openTime),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			CloseTime: int64(closeTime),
		})
	}
	return klines
}

func ExtractClosePrices(klines []types.Kline) []float64 {
	prices := make([]float64, len(klines))
	for i, k := range klines {
		prices[i] = k.Close
	}
	return prices
}

func ExtractVolumes(klines []types.Kline) []float64 {
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		volumes[i] = k.Volume
	}
	return volumes
}

func ComputeRSI(closes []float64, period int) []float64 {
	var rsi []float64
	for i := period; i < len(closes); i++ {
		var gain, loss float64
		for j := i - period + 1; j <= i; j++ {
			diff := closes[j] - closes[j-1]
			if diff >= 0 {
				gain += diff
			} else {
				loss -= diff
			}
		}
		avgGain := gain / float64(period)
		avgLoss := loss / float64(period)
		rs := avgGain / (avgLoss + 1e-10)
		rsi = append(rsi, 100-(100/(1+rs)))
	}
	return rsi
}

func ComputeMACD(closes []float64, shortPeriod, longPeriod, signalPeriod int) ([]float64, []float64, []float64) {
	shortEMA := computeEMA(closes, shortPeriod)
	longEMA := computeEMA(closes, longPeriod)

	minLen := int(math.Min(float64(len(shortEMA)), float64(len(longEMA))))
	macdLine := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		macdLine[i] = shortEMA[i] - longEMA[i]
	}

	signalLine := computeEMA(macdLine, signalPeriod)
	histogram := make([]float64, len(signalLine))
	for i := range signalLine {
		histogram[i] = macdLine[i+len(macdLine)-len(signalLine)] - signalLine[i]
	}

	return macdLine, signalLine, histogram
}

func ComputeVolumeMA(volumes []float64, period int) float64 {
	if len(volumes) < period {
		return 0
	}
	var sum float64
	for i := len(volumes) - period; i < len(volumes); i++ {
		sum += volumes[i]
	}
	return sum / float64(period)
}

func computeEMA(data []float64, period int) []float64 {
	var ema []float64
	k := 2.0 / (float64(period) + 1.0)
	for i := 0; i < len(data); i++ {
		if i < period {
			continue
		}
		if len(ema) == 0 {
			var sum float64
			for j := i - period; j < i; j++ {
				sum += data[j]
			}
			ema = append(ema, sum/float64(period))
		} else {
			prev := ema[len(ema)-1]
			ema = append(ema, (data[i]-prev)*k+prev)
		}
	}
	return ema
}

func ComputeATR(klines []types.Kline, period int) float64 {
	if len(klines) < period+1 {
		return 0
	}
	var trs []float64
	for i := 1; i < len(klines); i++ {
		high := klines[i].High
		low := klines[i].Low
		closePrev := klines[i-1].Close

		tr := math.Max(high-low, math.Max(math.Abs(high-closePrev), math.Abs(low-closePrev)))
		trs = append(trs, tr)
	}

	var sum float64
	for i := len(trs) - period; i < len(trs); i++ {
		sum += trs[i]
	}
	return sum / float64(period)
}
