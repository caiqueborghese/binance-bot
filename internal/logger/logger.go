package logger

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"
)

func LogTrade(symbol, side string, qty, price, saldo float64) {
	file, err := os.OpenFile("trades.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{
		time.Now().Format(time.RFC3339),
		symbol,
		side,
		formatFloat(qty),
		formatFloat(price),
		formatFloat(saldo),
	})
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.6f", f)
}
