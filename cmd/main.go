// main.go atualizado com lógica de PnL precisa e controle de posições
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
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

func getPositionInfo(apiKey, apiSecret, symbol string, leverage float64) (bool, float64, string, float64, float64, error) {
	// 1. Obtem o preço de marca (Mark Price)
	markPriceURL := fmt.Sprintf("https://fapi.binance.com/fapi/v1/premiumIndex?symbol=%s", symbol)
	resp, err := http.Get(markPriceURL)
	if err != nil {
		return false, 0, "", 0, 0, fmt.Errorf("erro ao obter mark price: %v", err)
	}
	defer resp.Body.Close()
	var markData struct {
		MarkPrice string `json:"markPrice"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&markData); err != nil {
		return false, 0, "", 0, 0, fmt.Errorf("erro ao decodificar mark price: %v", err)
	}
	markPrice, _ := strconv.ParseFloat(markData.MarkPrice, 64)

	// 2. Obtem os dados da posição
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params := "timestamp=" + timestamp
	signature := binance.Sign(params, apiSecret)
	url := fmt.Sprintf("https://fapi.binance.com/fapi/v2/positionRisk?%s&signature=%s", params, signature)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0, "", 0, 0, err
	}
	req.Header.Set("X-MBX-APIKEY", apiKey)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return false, 0, "", 0, 0, err
	}
	defer resp.Body.Close()

	var data []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false, 0, "", 0, 0, err
	}

	for _, item := range data {
		if item["symbol"] == symbol {
			posAmt, _ := strconv.ParseFloat(item["positionAmt"].(string), 64)
			if posAmt == 0 {
				return false, 0, "", 0, 0, nil
			}
			entry, _ := strconv.ParseFloat(item["entryPrice"].(string), 64)

			// 3. Cálculo do PnL com alavancagem
			var pnl float64
			if posAmt > 0 {
				pnl = (markPrice - entry) / entry * leverage * 100
			} else {
				pnl = (entry - markPrice) / entry * leverage * 100
			}

			side := "BUY"
			if posAmt < 0 {
				side = "SELL"
			}

			log.Printf("🔍 Position Debug | Side: %s | Qty: %.2f | Entry: %.4f | Mark: %.4f | PnL: %.2f%%",
				side, math.Abs(posAmt), entry, markPrice, pnl)

			return true, math.Abs(posAmt), side, entry, pnl, nil
		}
	}
	return false, 0, "", 0, 0, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
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

	symbol := "ETHUSDT"
	leverage := 20.0
	stepSizes := map[string]float64{
		"BTCUSDT": 0.001,
		"ETHUSDT": 0.01,
		"XRPUSDT": 0.1,
	}
	stepSize := stepSizes[symbol]

	for {
		saldo := client.GetUSDTBalance()
		fmt.Printf("\n💰 Saldo USDT: %.2f\n", saldo)

		klines := client.GetKlines(symbol, "1m", 100)
		closes := indicators.ExtractClosePrices(klines)
		volumes := indicators.ExtractVolumes(klines)
		macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
		rsi := indicators.ComputeRSI(closes, 14)
		volMA := indicators.ComputeVolumeMA(volumes, 14)

		currentPrice := client.GetMarkPrice(symbol)
		sig := strategy.EvaluateSignal(klines)

		inPosition, qty, side, entryPrice, pnl, err := getPositionInfo(apiKey, apiSecret, symbol, leverage)
		if err != nil {
			log.Printf("Erro ao buscar posição: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		fmt.Printf("📊 Posição — Entry: %.4f | Mark: %.4f | PnL: %.2f%%\n", entryPrice, currentPrice, pnl)

		if inPosition {
			fmt.Printf("📈 Em posição: %s | Qty: %.2f | Entrada: %.4f | PnL: %.2f%%\n", side, qty, entryPrice, pnl)

			if pnl >= 1.0 || pnl <= -1.0 {
				motivo := "TAKE PROFIT"
				if pnl <= -1.0 {
					motivo = "STOP LOSS"
				}

				if qty < stepSize {
					log.Printf("❌ Quantidade abaixo do mínimo (%s): %.4f < %.4f", symbol, qty, stepSize)
					time.Sleep(2 * time.Second)
					continue
				}

				closeSide := "SELL"
				if side == "SELL" {
					closeSide = "BUY"
				}

				msg := fmt.Sprintf("🔴 %s (%.2f%%) - Fechando posição %s Qty: %.1f", motivo, pnl, side, qty)
				fmt.Println(msg)

				ok := client.PlaceMarketOrder(symbol, closeSide, qty, true)
				if ok {
					telegram.SendMessage(msg)
					logger.LogTrade(symbol, motivo+"-CLOSE", qty, currentPrice, saldo)
				} else {
					fmt.Println("❌ Erro ao fechar posição!")
				}
				time.Sleep(2 * time.Second)
				continue
			}
			time.Sleep(2 * time.Second)
			continue
		}

		rawQty := saldo * 0.90 * leverage / currentPrice
		if rawQty < stepSize {
			log.Printf("❌ Quantidade insuficiente para %s (min: %.4f)", symbol, stepSize)
			time.Sleep(3 * time.Second)
			continue
		}
		orderQty := math.Floor(rawQty/stepSize) * stepSize

		switch sig {
		case strategy.BuySignal:
			side = "BUY"
		case strategy.SellSignal:
			side = "SELL"
		default:
			fmt.Println("⚪ Nenhum sinal — aguardando próximo candle...")
			time.Sleep(2 * time.Second)
			continue
		}

		msg := fmt.Sprintf("🟢 %s: %s | qty %.3f | alav %.0fx", side, symbol, orderQty, leverage)
		fmt.Println(msg)
		ok := client.PlaceMarketOrder(symbol, side, orderQty, false)
		if ok {
			msgDet := fmt.Sprintf(
				"%s\n\n📊 Indicadores:\n- MACD: %.4f / %.4f\n- RSI: %.2f\n- Volume: %.2f vs MA: %.2f\n💰 Preço: %.4f | Quantidade: %.1f | Saldo: %.2f",
				msg,
				macdLine[len(macdLine)-1],
				signalLine[len(signalLine)-1],
				rsi[len(rsi)-1],
				volumes[len(volumes)-1],
				volMA[len(volMA)-1],
				currentPrice,
				orderQty,
				saldo,
			)
			telegram.SendMessage(msgDet)
			logger.LogTrade(symbol, side, orderQty, currentPrice, saldo)
		} else {
			fmt.Println("❌ Erro ao executar ordem!")
		}

		time.Sleep(2 * time.Second)
	}
}
