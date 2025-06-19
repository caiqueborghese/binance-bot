// cmd/main.go
package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"

	"binance-bot/config"
	"binance-bot/internal/binance"
	"binance-bot/internal/indicators"
	"binance-bot/internal/logger"
	"binance-bot/internal/strategy"
	"binance-bot/internal/telegram"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env não encontrado, usando variáveis de ambiente")
	}

	apiKey := os.Getenv("BINANCE_API_KEY")
	apiSecret := os.Getenv("BINANCE_API_SECRET")
	if apiKey == "" || apiSecret == "" {
		log.Fatal("Faltando BINANCE_API_KEY ou BINANCE_API_SECRET")
	}

	cfg := config.Config{
		APIKey:    apiKey,
		APISecret: apiSecret,
		Testnet:   false,
	}
	client := binance.NewBinanceRestClient(cfg)

	symbol := "XRPUSDT"
	leverage := 20.0
	inPosition := false
	entryPrice := 0.0
	qty := 0.0
	trailingActive := false
	positionSide := ""
	trailingStartGain := 1.5
	trailingStopTrigger := 0.5

	for {
		saldo := client.GetUSDTBalance()
		fmt.Printf("\n💰 Saldo USDT: %.2f\n", saldo)

		klines := client.GetKlines(symbol, "1m", 100)
		closes := indicators.ExtractClosePrices(klines)
		price := closes[len(closes)-1]
		sig := strategy.EvaluateSignal(klines)

		if inPosition {
			gain := 0.0
			if positionSide == "BUY" {
				gain = (price - entryPrice) / entryPrice * 100 * leverage
			} else {
				gain = (entryPrice - price) / entryPrice * 100 * leverage
			}

			if trailingActive && gain <= trailingStopTrigger {
				msg := fmt.Sprintf("🔻 Trailing Stop acionado (%.2f%%) — Fechando posição.", gain)
				fmt.Println(msg)
				client.PlaceMarketOrder(symbol, "SELL", qty)
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "TRAIL-CLOSE", qty, price, saldo)
				inPosition = false
				trailingActive = false
			} else if gain >= 2.0 {
				msg := fmt.Sprintf("🎯 TAKE PROFIT atingido (%.2f%%)! Fechando posição.", gain)
				fmt.Println(msg)
				client.PlaceMarketOrder(symbol, "SELL", qty)
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "TP-CLOSE", qty, price, saldo)
				inPosition = false
				trailingActive = false
			} else if gain <= -3.0 {
				msg := fmt.Sprintf("⚠️ STOP LOSS ativado (%.2f%%)! Fechando posição.", gain)
				fmt.Println(msg)
				client.PlaceMarketOrder(symbol, "SELL", qty)
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "SL-CLOSE", qty, price, saldo)
				inPosition = false
				trailingActive = false
			} else {
				fmt.Printf("📊 Em posição %s - Variação atual: %.2f%%\n", positionSide, gain)
				if gain >= trailingStartGain {
					trailingActive = true
					fmt.Println("🔁 Trailing stop ativado!")
				}
			}
			time.Sleep(60 * time.Second)
			continue
		}

		rawQty := saldo * 0.95 * leverage / price
		if rawQty < 1.0 {
			log.Println("❌ Quantidade insuficiente para ordem mínima. Aguardando...")
			time.Sleep(60 * time.Second)
			continue
		}
		qty, _ = strconv.ParseFloat(fmt.Sprintf("%.3f", rawQty), 64)

		switch sig {
		case strategy.BuySignal:
			msg := fmt.Sprintf("🟢 COMPRA: %s | qty %.3f | alav %.0fx", symbol, qty, leverage)
			fmt.Println(msg)
			ok := client.PlaceMarketOrder(symbol, "BUY", qty)
			if ok {
				inPosition = true
				entryPrice = price
				positionSide = "BUY"
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "BUY", qty, price, saldo)
			} else {
				fmt.Println("❌ Erro ao executar ordem de compra!")
			}

		case strategy.SellSignal:
			msg := fmt.Sprintf("🔴 VENDA: %s | qty %.3f | alav %.0fx", symbol, qty, leverage)
			fmt.Println(msg)
			ok := client.PlaceMarketOrder(symbol, "SELL", qty)
			if ok {
				inPosition = true
				entryPrice = price
				positionSide = "SELL"
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "SELL", qty, price, saldo)
			} else {
				fmt.Println("❌ Erro ao executar ordem de venda!")
			}

		default:
			fmt.Println("⚪ Nenhum sinal — aguardando próximo candle...")
		}

		time.Sleep(60 * time.Second)
	}
}
