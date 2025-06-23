// cmd/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func fetchPositionPNL(apiKey, apiSecret, symbol string) (float64, float64, float64, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params := "timestamp=" + timestamp
	signature := binance.Sign(params, apiSecret)
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v2/positionRisk?%s&signature=%s", params, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, 0, 0, err
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, 0, err
	}
	defer resp.Body.Close()

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, 0, 0, err
	}

	for _, item := range data {
		if item["symbol"] == symbol {
			entryPrice, _ := strconv.ParseFloat(item["entryPrice"].(string), 64)
			markPrice, _ := strconv.ParseFloat(item["markPrice"].(string), 64)
			positionAmt, _ := strconv.ParseFloat(item["positionAmt"].(string), 64)

			if positionAmt == 0 {
				return 0, 0, 0, nil
			}

			var pnlPercent float64
			if positionAmt > 0 {
				pnlPercent = ((markPrice - entryPrice) / entryPrice) * 100
			} else {
				pnlPercent = ((entryPrice - markPrice) / entryPrice) * 100
			}

			return pnlPercent, entryPrice, markPrice, nil
		}
	}

	return 0, 0, 0, fmt.Errorf("posição não encontrada para %s", symbol)
}

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

	for {
		saldo := client.GetUSDTBalance()
		fmt.Printf("\n💰 Saldo USDT: %.2f\n", saldo)

		klines := client.GetKlines(symbol, "1m", 100)
		closes := indicators.ExtractClosePrices(klines)
		volumes := indicators.ExtractVolumes(klines)
		macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
		rsi := indicators.ComputeRSI(closes, 14)
		volMA := indicators.ComputeVolumeMA(volumes, 14)

		price := closes[len(closes)-1]
		sig := strategy.EvaluateSignal(klines)

		pnl, entry, mark, err := fetchPositionPNL(apiKey, apiSecret, symbol)
		if err != nil {
			log.Printf("Erro ao buscar PNL: %v\n", err)
		}
		fmt.Printf("📈 Verificando PnL: %.2f%% | Lucro: %.4f | Notional: %.2f\n", pnl, mark-entry, mark*leverage)

		if inPosition {
			if pnl >= 3.0 {
				msg := fmt.Sprintf("🎯 TAKE PROFIT atingido (+%.2f%%)! Fechando posição.", pnl)
				fmt.Println(msg)
				client.PlaceMarketOrder(symbol, "BUY", 100)
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "TP-CLOSE", 100, price, saldo)
				inPosition = false
			} else if pnl <= -1.0 {
				msg := fmt.Sprintf("⚠️ STOP LOSS ativado (%.2f%%)! Fechando posição.", pnl)
				fmt.Println(msg)
				client.PlaceMarketOrder(symbol, "BUY", 100)
				telegram.SendMessage(msg)
				logger.LogTrade(symbol, "SL-CLOSE", 100, price, saldo)
				inPosition = false
			} else {
				fmt.Printf("📊 Em posição - Variação atual: %.2f%%\n", pnl)
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
		qty, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", rawQty), 64)

		switch sig {
		case strategy.BuySignal:
			msg := fmt.Sprintf("🟢 COMPRA: %s | qty %.3f | alav %.0fx", symbol, qty, leverage)
			fmt.Println(msg)
			ok := client.PlaceMarketOrder(symbol, "BUY", qty)
			if ok {
				inPosition = true
				msgDet := fmt.Sprintf(
					"%s\n\n📊 Indicadores:\n- MACD: %.4f / %.4f\n- RSI: %.2f\n- Volume: %.2f vs Média: %.2f\n\n💰 Preço de entrada: %.4f\n🔁 Quantidade: %.1f\n⚙️ Alavancagem: %.0fx\n💼 Saldo USDT: %.2f\n\n⏱ Intervalo: 1m | Ativo: %s",
					msg,
					macdLine[len(macdLine)-1],
					signalLine[len(signalLine)-1],
					rsi[len(rsi)-1],
					volumes[len(volumes)-1],
					volMA[len(volMA)-1],
					price,
					qty,
					leverage,
					saldo,
					symbol,
				)
				telegram.SendMessage(msgDet)
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
				msgDet := fmt.Sprintf(
					"%s\n\n📊 Indicadores:\n- MACD: %.4f / %.4f\n- RSI: %.2f\n- Volume: %.2f vs Média: %.2f\n\n💰 Preço de entrada: %.4f\n🔁 Quantidade: %.1f\n⚙️ Alavancagem: %.0fx\n💼 Saldo USDT: %.2f\n\n⏱ Intervalo: 1m | Ativo: %s",
					msg,
					macdLine[len(macdLine)-1],
					signalLine[len(signalLine)-1],
					rsi[len(rsi)-1],
					volumes[len(volumes)-1],
					volMA[len(volMA)-1],
					price,
					qty,
					leverage,
					saldo,
					symbol,
				)
				telegram.SendMessage(msgDet)
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
