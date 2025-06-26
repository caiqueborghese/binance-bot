// main.go com trailing stop por PnL dinâmico e SL fixo de -5%, com precisão corrigida para ordens
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"binance-bot/config"
	"binance-bot/internal/binance"
	"binance-bot/internal/indicators"
	"binance-bot/internal/logger"
	"binance-bot/internal/strategy"
	"binance-bot/internal/telegram"
)

type TrailingStatus struct {
	MaxPnL float64
	Side   string
}

func countDecimals(step float64) int {
	str := fmt.Sprintf("%f", step)
	parts := strings.Split(str, ".")
	decimals := strings.TrimRight(parts[1], "0")
	return len(decimals)
}

func getPositionInfo(apiKey, apiSecret, symbol string, leverage float64) (bool, float64, string, float64, float64, error) {
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

			log.Printf("🔍 Position Debug | %s | Qty: %.2f | Entry: %.4f | Mark: %.4f | PnL: %.2f%%",
				side, math.Abs(posAmt), entry, markPrice, pnl)

			return true, math.Abs(posAmt), side, entry, pnl, nil
		}
	}
	return false, 0, "", 0, 0, nil
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

	symbols := []string{"ETHUSDT", "BTCUSDT", "XRPUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT", "MATICUSDT", "DOTUSDT", "AVAXUSDT", "LINKUSDT"}
	leverage := 20.0
	stepSizes := map[string]float64{
		"ETHUSDT": 0.01, "BTCUSDT": 0.001, "XRPUSDT": 0.1, "BNBUSDT": 0.01,
		"ADAUSDT": 1.0, "SOLUSDT": 0.01, "MATICUSDT": 1.0, "DOTUSDT": 0.1,
		"AVAXUSDT": 0.01, "LINKUSDT": 0.1,
	}

	trailings := make(map[string]*TrailingStatus)

	for {
		saldo := client.GetUSDTBalance()
		fmt.Printf("\n💰 Saldo USDT: %.2f\n", saldo)

		for _, symbol := range symbols {
			stepSize := stepSizes[symbol]
			rawKlines := client.GetKlines(symbol, "1m", 100)
			if len(rawKlines) == 0 {
				log.Printf("⚠️ Falha ao obter klines para %s, pulando...", symbol)
				continue
			}

			klines := indicators.ConvertToKlines(rawKlines)
			closes := indicators.ExtractClosePrices(klines)
			volumes := indicators.ExtractVolumes(klines)
			macdLine, signalLine, _ := indicators.ComputeMACD(closes, 12, 26, 9)
			rsi := indicators.ComputeRSI(closes, 14)
			volMA := indicators.ComputeVolumeMA(volumes, 14)
			currentPrice := client.GetMarkPrice(symbol)

			sig := strategy.EvaluateSignal(klines, symbol)
			inPosition, qty, side, _, pnl, err := getPositionInfo(apiKey, apiSecret, symbol, leverage)
			if err != nil {
				log.Printf("Erro ao buscar posição para %s: %v\n", symbol, err)
				continue
			}

			if inPosition {
				trailing, exists := trailings[symbol]
				if !exists {
					trailings[symbol] = &TrailingStatus{MaxPnL: pnl, Side: side}
					continue
				}
				if pnl > trailing.MaxPnL {
					trailing.MaxPnL = pnl
				}

				shouldExit := false
				if trailing.MaxPnL >= 3.0 && pnl <= trailing.MaxPnL-1.0 {
					shouldExit = true
				}
				if pnl <= -5.0 {
					shouldExit = true
				}

				if shouldExit {
					closeSide := "SELL"
					if trailing.Side == "SELL" {
						closeSide = "BUY"
					}
					if qty < stepSize {
						log.Printf("❌ Quantidade abaixo do mínimo (%s): %.4f < %.4f", symbol, qty, stepSize)
						continue
					}
					saldoAntes := client.GetUSDTBalance()
					ok := client.PlaceMarketOrder(symbol, closeSide, qty, true)
					if ok {
						time.Sleep(1 * time.Second)
						saldoDepois := client.GetUSDTBalance()
						lucroReal := saldoDepois - saldoAntes
						msg := fmt.Sprintf("🔴 %s (MaxPnL %.2f%% → %.2f%%) Fechando %s Qty: %.3f", symbol, trailing.MaxPnL, pnl, trailing.Side, qty)
						msgLucro := fmt.Sprintf("🔎 Lucro real: %.4f USDT", lucroReal)
						telegram.SendMessage(msg + "\n" + msgLucro)
						logger.LogTrade(symbol, "TRAILING-CLOSE", qty, currentPrice, saldoDepois)
						delete(trailings, symbol)
					}
				}
				continue
			}

			rawQty := saldo * 0.90 * leverage / currentPrice
			if rawQty < stepSize {
				log.Printf("❌ Quantidade insuficiente para %s (min: %.4f)", symbol, stepSize)
				continue
			}
			decimals := countDecimals(stepSize)
			factor := math.Pow(10, float64(decimals))
			orderQty := math.Floor(rawQty*factor) / factor

			var orderSide string
			switch sig {
			case strategy.BuySignal:
				orderSide = "BUY"
			case strategy.SellSignal:
				orderSide = "SELL"
			default:
				fmt.Printf("⚪ %s: Nenhum sinal válido\n", symbol)
				continue
			}

			saldoAntes := client.GetUSDTBalance()
			msg := fmt.Sprintf("🟢 %s %s | qty %.3f | alav %.0fx", orderSide, symbol, orderQty, leverage)
			fmt.Println(msg)
			ok := client.PlaceMarketOrder(symbol, orderSide, orderQty, false)
			if ok {
				time.Sleep(1 * time.Second)
				saldoDepois := client.GetUSDTBalance()
				custo := saldoAntes - saldoDepois
				msgDet := fmt.Sprintf("%s\n\n📊 Indicadores:\n- MACD: %.4f / %.4f\n- RSI: %.2f\n- Volume: %.2f vs MA: %.2f\n💰 Preço: %.4f | Quantidade: %.1f | Custo: %.4f | Saldo: %.2f",
					msg,
					macdLine[len(macdLine)-1],
					signalLine[len(signalLine)-1],
					rsi[len(rsi)-1],
					volumes[len(volumes)-1],
					volMA,
					currentPrice,
					orderQty,
					custo,
					saldoDepois)
				telegram.SendMessage(msgDet)
				logger.LogTrade(symbol, orderSide, orderQty, currentPrice, saldoDepois)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
