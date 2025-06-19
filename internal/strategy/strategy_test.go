// internal/strategy/strategy_test.go
package strategy

import (
	"fmt"
	"testing"
)

// montarKlines constrói klines sintéticos dados slices de close e volume
func montarKlines(closes, vols []float64) [][]interface{} {
	res := make([][]interface{}, len(closes))
	for i := range closes {
		// [openTime, open, high, low, close, volume]
		res[i] = []interface{}{0, "0", 0, 0, fmt.Sprintf("%f", closes[i]), fmt.Sprintf("%f", vols[i])}
	}
	return res
}

func TestEvaluateSignal_Buy(t *testing.T) {
	// Prepara dados: macd > signal, rsi ~50, volume > MA
	closes := []float64{1, 1.1, 1.2, 1.3, 1.4}
	vols := []float64{10, 12, 14, 16, 18}
	klines := montarKlines(closes, vols)

	signal := EvaluateSignal(klines)
	if signal != BuySignal {
		t.Errorf("EvaluateSignal = %v; want BuySignal", signal)
	}
}

func TestEvaluateSignal_Sell(t *testing.T) {
	closes := []float64{1, 0.9, 0.8, 0.7, 0.6}
	vols := []float64{18, 16, 14, 12, 10}
	klines := montarKlines(closes, vols)

	signal := EvaluateSignal(klines)
	if signal != SellSignal {
		t.Errorf("EvaluateSignal = %v; want SellSignal", signal)
	}
}

func TestEvaluateSignal_NoSignal(t *testing.T) {
	closes := []float64{1, 1, 1, 1, 1}
	vols := []float64{10, 10, 10, 10, 10}
	klines := montarKlines(closes, vols)

	signal := EvaluateSignal(klines)
	if signal != NoSignal {
		t.Errorf("EvaluateSignal = %v; want NoSignal", signal)
	}
}
