package indicators

import (
	"strconv"
)

// ExtractClosePrices converte um slice de klines ([[openTime, open, high, low, close, volume, ...]])
// em um slice de preços de fechamento.
func ExtractClosePrices(klines [][]interface{}) []float64 {
	closes := make([]float64, len(klines))
	for i, k := range klines {
		// k[4] é fechamento como string
		if s, ok := k[4].(string); ok {
			val, _ := strconv.ParseFloat(s, 64)
			closes[i] = val
		}
	}
	return closes
}

// ExtractVolumes converte um slice de klines em um slice de volumes.
func ExtractVolumes(klines [][]interface{}) []float64 {
	vols := make([]float64, len(klines))
	for i, k := range klines {
		// k[5] é volume como string
		if s, ok := k[5].(string); ok {
			val, _ := strconv.ParseFloat(s, 64)
			vols[i] = val
		}
	}
	return vols
}

// ComputeEMA calcula a Média Móvel Exponencial para um slice de preços e período dado.
func ComputeEMA(prices []float64, period int) []float64 {
	ema := make([]float64, len(prices))
	if period <= 0 || len(prices) < period {
		return ema
	}

	mult := 2.0 / float64(period+1)
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*mult + ema[i-1]
	}
	return ema
}

// ComputeMACD calcula a linha MACD, a linha de sinal e o histograma.
func ComputeMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (macdLine, signalLine, histogram []float64) {
	fastEMA := ComputeEMA(prices, fastPeriod)
	slowEMA := ComputeEMA(prices, slowPeriod)
	length := len(prices)

	macdLine = make([]float64, length)
	for i := 0; i < length; i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	signalLine = ComputeEMA(macdLine, signalPeriod)

	histogram = make([]float64, length)
	for i := 0; i < length; i++ {
		histogram[i] = macdLine[i] - signalLine[i]
	}

	return macdLine, signalLine, histogram
}

// ComputeRSI calcula o Índice de Força Relativa (RSI) para um slice de preços e período.
func ComputeRSI(prices []float64, period int) []float64 {
	rsi := make([]float64, len(prices))
	if period <= 0 || len(prices) <= period {
		return rsi
	}

	gains := make([]float64, len(prices))
	losses := make([]float64, len(prices))
	for i := 1; i < len(prices); i++ {
		delta := prices[i] - prices[i-1]
		if delta > 0 {
			gains[i] = delta
		} else {
			losses[i] = -delta
		}
	}

	sumGain, sumLoss := 0.0, 0.0
	for i := 1; i <= period; i++ {
		sumGain += gains[i]
		sumLoss += losses[i]
	}
	rs := sumGain / sumLoss
	rsi[period] = 100 - (100 / (1 + rs))

	for i := period + 1; i < len(prices); i++ {
		sumGain = (sumGain*float64(period-1) + gains[i]) / float64(period)
		sumLoss = (sumLoss*float64(period-1) + losses[i]) / float64(period)
		rs = sumGain / sumLoss
		rsi[i] = 100 - (100 / (1 + rs))
	}

	return rsi
}

// ComputeVolumeMA calcula a Média Móvel Simples do volume.
func ComputeVolumeMA(volumes []float64, period int) []float64 {
	ma := make([]float64, len(volumes))
	if period <= 0 || len(volumes) < period {
		return ma
	}

	sum := 0.0
	for i := 0; i < period; i++ {
		sum += volumes[i]
	}
	ma[period-1] = sum / float64(period)

	for i := period; i < len(volumes); i++ {
		sum += volumes[i] - volumes[i-period]
		ma[i] = sum / float64(period)
	}
	return ma
}

// --- NOVAS FUNÇÕES AUXILIARES ---

// Último valor do MACD
func LastMACD(prices []float64, fast, slow, signal int) (macd, signalLine, hist float64) {
	m, s, h := ComputeMACD(prices, fast, slow, signal)
	last := len(prices) - 1
	return m[last], s[last], h[last]
}

// Último valor do RSI
func LastRSI(prices []float64, period int) float64 {
	r := ComputeRSI(prices, period)
	return r[len(prices)-1]
}

// Último valor da média de volume
func LastVolumeMA(vols []float64, period int) float64 {
	v := ComputeVolumeMA(vols, period)
	return v[len(vols)-1]
}

// Extrai os preços de abertura
func ExtractOpenPrices(klines [][]interface{}) []float64 {
	opens := make([]float64, len(klines))
	for i, k := range klines {
		if s, ok := k[1].(string); ok {
			val, _ := strconv.ParseFloat(s, 64)
			opens[i] = val
		}
	}
	return opens
}

// Verifica se a última vela está em tamanho razoável
func IsCandleReasonable(klines [][]interface{}, lookback int, maxFactor float64) bool {
	if len(klines) < lookback+1 {
		return true
	}

	last := len(klines) - 1
	var sum float64
	for i := last - lookback; i < last; i++ {
		high := parseStrToFloat(klines[i][2])
		low := parseStrToFloat(klines[i][3])
		sum += high - low
	}
	avgRange := sum / float64(lookback)

	// Faixa da última vela
	hlLast := parseStrToFloat(klines[last][2]) - parseStrToFloat(klines[last][3])

	return hlLast <= avgRange*maxFactor
}

func parseStrToFloat(val interface{}) float64 {
	if s, ok := val.(string); ok {
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}
