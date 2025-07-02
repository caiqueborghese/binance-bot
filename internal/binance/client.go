package binance

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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
	base := "https://fapi.binance.com"
	if cfg.Testnet {
		base = "https://testnet.binancefuture.com"
	}
	return &BinanceRestClient{
		APIKey:    cfg.APIKey,
		APISecret: cfg.APISecret,
		BaseURL:   base,
	}
}

func Sign(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

type OrderRequest struct {
	Symbol      string `json:"symbol"`
	Side        string `json:"side"`
	Type        string `json:"type"`
	Quantity    string `json:"quantity"`
	RecvWindow  int    `json:"recvWindow"`
	Timestamp   int64  `json:"timestamp"`
	ReduceOnly  bool   `json:"reduceOnly,omitempty"`
	NewClientID string `json:"newClientOrderId,omitempty"`
}

func (b *BinanceRestClient) PlaceMarketOrder(symbol, side string, quantity float64, reduceOnly bool) bool {
	endpoint := "/fapi/v1/order"
	url := b.BaseURL + endpoint

	quantityStr := strconv.FormatFloat(quantity, 'f', -1, 64)
	order := OrderRequest{
		Symbol:     symbol,
		Side:       side,
		Type:       "MARKET",
		Quantity:   quantityStr,
		RecvWindow: 5000,
		Timestamp:  time.Now().UnixMilli(),
		ReduceOnly: reduceOnly,
	}

	body, _ := json.Marshal(order)
	params := fmt.Sprintf("symbol=%s&side=%s&type=MARKET&quantity=%s&recvWindow=5000&timestamp=%d",
		symbol, side, quantityStr, order.Timestamp)
	if reduceOnly {
		params += "&reduceOnly=true"
	}
	signature := Sign(params, b.APISecret)
	req, err := http.NewRequest("POST", url+"?"+params+"&signature="+signature, bytes.NewBuffer(body))
	if err != nil {
		log.Println("Erro ao criar requisição:", err)
		return false
	}
	req.Header.Set("X-MBX-APIKEY", b.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Erro ao enviar requisição:", err)
		return false
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if code, ok := result["code"]; ok {
		log.Printf("📨 Order response: %+v", result)
		log.Printf("❌ Erro da Binance: code %v, msg: %v", code, result["msg"])
		return false
	}
	log.Printf("📨 Ordem executada com sucesso: %+v", result)
	return true
}

func (b *BinanceRestClient) GetUSDTBalance() float64 {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params := "timestamp=" + timestamp
	signature := Sign(params, b.APISecret)
	url := b.BaseURL + "/fapi/v2/account?" + params + "&signature=" + signature

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-MBX-APIKEY", b.APIKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Erro ao obter saldo:", err)
		return 0.0
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if available, ok := result["availableBalance"].(string); ok {
		balance, _ := strconv.ParseFloat(available, 64)
		return balance
	}
	return 0.0
}

func (b *BinanceRestClient) GetMarkPrice(symbol string) float64 {
	url := b.BaseURL + "/fapi/v1/premiumIndex?symbol=" + symbol
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Erro ao obter mark price:", err)
		return 0.0
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	price, _ := strconv.ParseFloat(result["markPrice"].(string), 64)
	return price
}

func (b *BinanceRestClient) GetKlines(symbol, interval string, limit int) [][]interface{} {
	url := fmt.Sprintf("%s/fapi/v1/klines?symbol=%s&interval=%s&limit=%d", b.BaseURL, symbol, interval, limit)
	resp, err := http.Get(url)
	if err != nil {
		log.Println("Erro ao obter klines:", err)
		return nil
	}
	defer resp.Body.Close()

	var klines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&klines); err != nil {
		log.Println("Error parsing klines JSON:", err)
		return nil
	}
	return klines
}
