package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	APIKey    string
	APISecret string
	Testnet   bool
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Arquivo .env não encontrado. Usando variáveis do sistema.")
	}

	return Config{
		APIKey:    os.Getenv("BINANCE_API_KEY"),
		APISecret: os.Getenv("BINANCE_API_SECRET"),
		Testnet:   os.Getenv("BINANCE_TESTNET") == "true",
	}
}
