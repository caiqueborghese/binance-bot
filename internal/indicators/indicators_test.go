// internal/indicators/indicators_test.go
package indicators

import (
	"reflect"
	"testing"
)

func TestExtractClosePrices(t *testing.T) {
	klines := [][]interface{}{
		{0, "1.0", 0, 0, "1.1", 0},
		{0, "2.0", 0, 0, "2.2", 0},
		{0, "3.0", 0, 0, "3.3", 0},
	}
	expected := []float64{1.1, 2.2, 3.3}
	result := ExtractClosePrices(klines)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractClosePrices = %v; want %v", result, expected)
	}
}

func TestExtractVolumes(t *testing.T) {
	klines := [][]interface{}{
		{0, 0, 0, 0, 0, "10.0"},
		{0, 0, 0, 0, 0, "20.0"},
	}
	expected := []float64{10.0, 20.0}
	result := ExtractVolumes(klines)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ExtractVolumes = %v; want %v", result, expected)
	}
}

func TestComputeEMA(t *testing.T) {
	prices := []float64{1, 2, 3, 4, 5}
	period := 3
	ema := ComputeEMA(prices, period)
	// Check length and first non-zero index
	if len(ema) != len(prices) {
		t.Fatalf("ComputeEMA length = %d; want %d", len(ema), len(prices))
	}
	// The 2nd index (period-1) should equal SMA of first 3 elements = (1+2+3)/3 = 2
	if ema[period-1] != 2 {
		t.Errorf("ComputeEMA at index %d = %v; want %v", period-1, ema[period-1], 2.0)
	}
}

func TestComputeMACD(t *testing.T) {
	prices := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	macdLine, signalLine, histogram := ComputeMACD(prices, 2, 5, 3)
	if len(macdLine) != len(prices) || len(signalLine) != len(prices) || len(histogram) != len(prices) {
		t.Errorf("ComputeMACD output lengths mismatch")
	}
}

func TestComputeRSI(t *testing.T) {
	prices := []float64{1, 2, 1, 2, 1, 2, 1}
	rsi := ComputeRSI(prices, 3)
	// RSI values should be between 0 and 100
	for i, v := range rsi {
		if v < 0 || v > 100 {
			t.Errorf("RSI[%d] = %v out of range [0,100]", i, v)
		}
	}
}

func TestComputeVolumeMA(t *testing.T) {
	vols := []float64{1, 2, 3, 4, 5}
	ma := ComputeVolumeMA(vols, 2)
	if len(ma) != len(vols) {
		t.Fatalf("ComputeVolumeMA length = %d; want %d", len(ma), len(vols))
	}
	// Check at index 1: average of vols[0:2] = 1.5
	if ma[1] != 1.5 {
		t.Errorf("ComputeVolumeMA[1] = %v; want 1.5", ma[1])
	}
}
