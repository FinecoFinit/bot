package main

import (
	"bot/service"
	_ "github.com/mattn/go-sqlite3"
	"os"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	c, err := service.ReadConfig(configPath)
	if err != nil {
		panic(err)
	}

	app, err := service.Initialize(c)
	if err != nil {
		panic(err)
	}
	go app.Telegram.RestoreSessions()
	go app.Telegram.RestoreTimedEnable()
	app.Telegram.Tg.Start()
}
