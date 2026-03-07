package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file loaded")
	}
}

func GetFrontendURL() string {
	url := os.Getenv("FRONTEND_URL")

	return url
}
