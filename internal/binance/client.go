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

func NewBinanceRestClient(cfg config.Config) *BinanceRestClient {
	return &BinanceRestClient{
		APIKey:    cfg.APIKey,
		APISecret: cfg.APISecret,
		BaseURL:   "https://fapi.binance.com",
	}
}

func (c *BinanceRestClient) sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.APISecret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

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

func roundQuantity(qty, stepSize float64) float64 {
	return math.Floor(qty/stepSize) * stepSize
}

func (c *BinanceRestClient) PlaceMarketOrder(symbol, side string, quantity float64, reduceOnly bool) bool {
	stepSizes := map[string]float64{
		"BTCUSDT": 0.001,
		"ETHUSDT": 0.01,
		"XRPUSDT": 0.1,
	}
	stepSize, ok := stepSizes[symbol]
	if !ok {
		stepSize = 0.001
		log.Printf("⚠️ Usando stepSize padrão para %s: %.4f", symbol, stepSize)
	}

	qty := math.Floor(quantity/stepSize) * stepSize
	if qty <= 0 {
		log.Printf("❌ Quantity %.6f abaixo do stepSize %.6f", quantity, stepSize)
		return false
	}

	precision := int(math.Round(-math.Log10(stepSize)))
	if precision < 0 {
		precision = 0
	}

	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	rw := "5000"
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", "MARKET")
	params.Set("quantity", fmt.Sprintf("%.*f", precision, qty))
	params.Set("timestamp", ts)
	params.Set("recvWindow", rw)
	params.Set("newOrderRespType", "RESULT")

	if reduceOnly {
		params.Set("reduceOnly", "true")
		log.Printf("🔁 Enviando ordem para fechar posição (reduceOnly)")
	}

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
		log.Printf("❌ Erro ao decodificar resposta: %v", err)
		return false
	}

	if code, ok := result["code"].(float64); ok && code != 0 {
		log.Printf("❌ Erro da Binance: code %.0f, msg: %s", code, result["msg"])
		return false
	}

	log.Printf("✅ Ordem executada com sucesso! Symbol: %s, Side: %s, Qty: %.4f", symbol, side, qty)
	return true
}

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

func (c *BinanceRestClient) GetMarkPrice(symbol string) float64 {
	endpoint := fmt.Sprintf("%s/fapi/v1/premiumIndex?symbol=%s", c.BaseURL, symbol)
	resp, err := http.Get(endpoint)
	if err != nil {
		log.Printf("❌ Erro ao buscar mark price: %v", err)
		return 0.0
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data struct {
		MarkPrice string `json:"markPrice"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		log.Printf("❌ Erro ao decodificar mark price: %v", err)
		return 0.0
	}
	price, _ := strconv.ParseFloat(data.MarkPrice, 64)
	return price
}

// ✅ Função auxiliar de assinatura para posição
func Sign(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
