// internal/binance/client.go
package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"binance-bot/config"
)

type BinanceRestClient struct {
	APIKey    string
	APISecret string
	BaseURL   string
}

// NewBinanceRestClient retorna um cliente configurado para Binance Futures Mainnet
func NewBinanceRestClient(cfg config.Config) *BinanceRestClient {
	return &BinanceRestClient{
		APIKey:    cfg.APIKey,
		APISecret: cfg.APISecret,
		BaseURL:   "https://fapi.binance.com",
	}
}

// sign gera a assinatura HMAC SHA256 necessária para endpoints private
func (c *BinanceRestClient) sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// GetUSDTBalance busca o saldo de USDT na Binance Futures
func (c *BinanceRestClient) GetUSDTBalance() float64 {
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	rw := "5000"
	params := url.Values{}
	params.Set("timestamp", ts)
	params.Set("recvWindow", rw)
	q := params.Encode()
	sig := c.sign(q)
	url := fmt.Sprintf("%s/fapi/v2/balance?%s&signature=%s", c.BaseURL, q, sig)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Error creating balance request: %v", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Error fetching balance: %v", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var arr []struct{ Asset, Balance string }
	if err := json.Unmarshal(body, &arr); err != nil {
		log.Fatalf("Error parsing balance JSON: %v", err)
	}
	for _, a := range arr {
		if a.Asset == "USDT" {
			v, _ := strconv.ParseFloat(a.Balance, 64)
			return v
		}
	}
	return 0.0
}

// roundQuantity floors qty to the nearest multiple of stepSize
func roundQuantity(qty, stepSize float64) float64 {
	return math.Floor(qty/stepSize) * stepSize
}

// PlaceMarketOrder envia ordem MARKET (BUY ou SELL) e retorna sucesso/falha
func (c *BinanceRestClient) PlaceMarketOrder(symbol, side string, quantity float64) bool {
	stepSizes := map[string]float64{
		"BTCUSDT": 0.001,
		"ETHUSDT": 0.01,
		"XRPUSDT": 0.1,
	}
	stepSize, ok := stepSizes[symbol]
	if !ok {
		stepSize = 0.001
	}
	qty := roundQuantity(quantity, stepSize)
	if qty <= 0 {
		log.Printf("❌ Quantity %.6f abaixo do stepSize %.6f", quantity, stepSize)
		return false
	}
	precision := int(math.Round(-math.Log10(stepSize)))

	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	rw := "5000"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("quantity", fmt.Sprintf("%.*f", precision, qty))
	params.Set("timestamp", ts)
	params.Set("recvWindow", rw)
	q := params.Encode()
	sig := c.sign(q)
	url := fmt.Sprintf("%s/fapi/v1/order?%s&signature=%s", c.BaseURL, q, sig)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Printf("❌ Erro criando request: %v", err)
		return false
	}
	req.Header.Set("X-MBX-APIKEY", c.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Erro enviando ordem: %v", err)
		return false
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	log.Printf("📨 Order response: %s", string(body))

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return false
	}
	if code, ok := result["code"].(float64); ok && code != 0 {
		return false
	}
	return true
}

// GetKlines retorna os últimos candles do par especificado
func (c *BinanceRestClient) GetKlines(symbol, interval string, limit int) [][]interface{} {
	endpoint := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", c.BaseURL, symbol, interval, limit)
	resp, err := http.Get(endpoint)
	if err != nil {
		log.Fatalf("Error fetching klines: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var klines [][]interface{}
	if err := json.Unmarshal(body, &klines); err != nil {
		log.Fatalf("Error parsing klines JSON: %v", err)
	}
	return klines
}
