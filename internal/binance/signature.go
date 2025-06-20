// internal/binance/signature.go
package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// Sign gera a assinatura HMAC-SHA256 dos parâmetros usando a chave secreta.
func Sign(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
