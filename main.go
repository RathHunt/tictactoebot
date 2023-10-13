package main

import (
	"log"
	"os"
	handler "tictactoe/api"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")

	if botToken == "" {
		log.Fatal("Bot Token env must be set")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Bot is now running. Press Ctrl+C to exit.")

	for update := range updates {
		if update.Message != nil {
			handler.HandleTextMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			handler.HandleCallbackQuery(bot, update.CallbackQuery)
		} else if update.InlineQuery != nil {
			handler.HandleInlineQuery(bot, update.InlineQuery)
		}
	}
}
