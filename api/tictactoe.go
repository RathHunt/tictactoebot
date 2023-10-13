package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"tictactoe/game"

	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/joho/godotenv"
)

var bot *tgbotapi.BotAPI
var ctx = context.TODO()
var redisClient *redis.Client

func init() {
	godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Bot Token env must be set")
	}

	_bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	} else {
		bot = _bot
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		log.Fatal("REDIS_ADDR env must be set")
	}
	redisOptions, err := redis.ParseURL(redisAddr)
	if err != nil {
		log.Fatal("Error parsing Redis connection string:", err)
	}

	redisClient = redis.NewClient(redisOptions)
}

func TicTacToeHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var update tgbotapi.Update
	err = json.Unmarshal(body, &update)
	if err != nil {
		http.Error(w, "Error unmarshalling update", http.StatusInternalServerError)
		return
	}

	if update.Message != nil {
		HandleTextMessage(bot, update.Message)
		log.Println("got text message")
	} else if update.CallbackQuery != nil {
		HandleCallbackQuery(bot, update.CallbackQuery)
		log.Println("got callback query")
	} else if update.InlineQuery != nil {
		HandleInlineQuery(bot, update.InlineQuery)
		log.Println("got inline query")
	}

	fmt.Fprintf(w, "Received a message")
}

func HandleTextMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	log.Println("HandleTextMessage called")

	replyText := "Hello! Let's play Tic Tac Toe."

	if message.Chat.Type == "private" {
		msg := tgbotapi.NewMessage(message.Chat.ID, replyText)
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonSwitch("Start Game", ""),
			),
		)
		_, err := bot.Send(msg)
		if err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

func HandleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
	log.Println("HandleCallbackQuery called")

	var gameId, row, col int
	fmt.Sscanf(query.Data, "%d_%d_%d", &gameId, &row, &col)

	// Retrieve game state from Redis
	key := fmt.Sprintf("game:%d", gameId)
	gameStateJSON, err := redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		log.Println("Game state not found for ID:", gameId)
		return
	} else if err != nil {
		log.Println("Error retrieving game state from Redis:", err)
		return
	}

	var game game.TicTacToeGame
	err = json.Unmarshal([]byte(gameStateJSON), &game)
	if err != nil {
		log.Println("Error unmarshalling game state:", err)
		return
	}

	// Process move, update game state, and store it back in Redis
	callbackConfig := game.ProcessMove(bot, query)
	bot.AnswerCallbackQuery(callbackConfig)

	data, err := json.Marshal(game)
	if err != nil {
		log.Println("Error marshalling game state:", err)
	}

	gameStateJSONString := string(data)

	err = redisClient.Set(ctx, key, gameStateJSONString, 0).Err()
	if err != nil {
		log.Println("Error storing updated game state in Redis:", err)
		return
	}

	messageText := game.GetMessageText()
	replyMarkup := game.GetKeyBoard()
	editConfig := tgbotapi.NewEditMessageText(0, 0, messageText)
	editConfig.BaseEdit.InlineMessageID = query.InlineMessageID
	editConfig.ReplyMarkup = &replyMarkup

	_, err = bot.Send(editConfig)
	if err != nil {
		log.Println("Error editing inline message:", err)
	}
}

func generateUniqueID() int {
	// Increment a Redis key to get the next sequential ID
	nextID, err := redisClient.Incr(ctx, "game_id_counter").Result()
	if err != nil {
		log.Fatal("Error incrementing game ID counter:", err)
	}

	return int(nextID)
}

func HandleInlineQuery(bot *tgbotapi.BotAPI, inlineQuery *tgbotapi.InlineQuery) {
	if inlineQuery.From.IsBot || inlineQuery.From.ID == bot.Self.ID {
		return
	}

	newGameID := generateUniqueID() // Implement your own function to generate a unique ID
	newGame := game.NewGame(inlineQuery.From, newGameID)

	// Store the new game in Redis
	gameStateJSON, err := json.Marshal(newGame)
	if err != nil {
		log.Println("Error marshalling new game state:", err)
		return
	}

	key := fmt.Sprintf("game:%d", newGameID)
	err = redisClient.Set(ctx, key, gameStateJSON, 0).Err()
	if err != nil {
		log.Println("Error storing new game state in Redis:", err)
		return
	}

	messageText := newGame.GetMessageText()
	replyMarkup := newGame.GetKeyBoard()

	article := tgbotapi.NewInlineQueryResultArticle(inlineQuery.ID, "Start Tic Tac Toe Game", messageText)
	article.ReplyMarkup = &replyMarkup

	response := tgbotapi.InlineConfig{
		InlineQueryID: inlineQuery.ID,
		Results:       []interface{}{article},
	}

	_, err = bot.AnswerInlineQuery(response)
	if err != nil {
		log.Println("Error answering inline query:", err)
	}
}
