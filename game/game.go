package game

import (
	"encoding/json"
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

const (
	Host gameSpace = iota
	Guest
	Null
)

type gameSpace int

func (btn *gameSpace) getSymbol() string {
	switch *btn {
	case Host:
		return "❎"
	case Guest:
		return "⭕"
	default:
		return " "
	}
}

type TicTacToeGame struct {
	Grid        [3][3]gameSpace   `json:"grid"`
	Players     [2]*tgbotapi.User `json:"players"`
	CurrentTurn int               `json:"currentTurn"`
	GameID      int               `json:"gameID"`
}

func NewGame(creator *tgbotapi.User, gameID int) *TicTacToeGame {
	game := &TicTacToeGame{
		Grid:        [3][3]gameSpace{{Null, Null, Null}, {Null, Null, Null}, {Null, Null, Null}},
		Players:     [2]*tgbotapi.User{creator, nil},
		CurrentTurn: 1,
		GameID:      gameID,
	}

	return game
}

func (game *TicTacToeGame) GetMessageText() string {
	if game.Players[1] == nil {
		return fmt.Sprintf("Waiting for the second player to join...\n\n%s is waiting for an opponent.", game.Players[0].UserName)
	}

	if game.checkForWin() {
		winner := game.Players[(game.CurrentTurn)%2].UserName
		return fmt.Sprintf("🎉 %s wins! 🎉\n\n%s vs %s", winner, game.Players[0].UserName, game.Players[1].UserName)
	} else if game.checkForTie() {
		return fmt.Sprintf("It's a tie!\n\n%s vs %s", game.Players[0].UserName, game.Players[1].UserName)
	} else {
		currentPlayer := game.Players[game.CurrentTurn%2].UserName
		return fmt.Sprintf("It's %s's turn\n\n%s vs %s", currentPlayer, game.Players[0].UserName, game.Players[1].UserName)
	}
}

func (game *TicTacToeGame) GetKeyBoard() tgbotapi.InlineKeyboardMarkup {
	var buttons []tgbotapi.InlineKeyboardButton
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			btnText := game.Grid[row][col].getSymbol()
			callbackData := fmt.Sprintf("%d_%d_%d", game.GameID, row+1, col+1)
			btn := tgbotapi.NewInlineKeyboardButtonData(btnText, callbackData)
			buttons = append(buttons, btn)
		}
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		buttons[0:3],
		buttons[3:6],
		buttons[6:9],
	}

	inlineKeyboardMarkup := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return inlineKeyboardMarkup
}

func (game *TicTacToeGame) ProcessMove(
	bot *tgbotapi.BotAPI,
	query *tgbotapi.CallbackQuery,
) tgbotapi.CallbackConfig {
	data, _ := json.MarshalIndent(query, "", "    ")
	fmt.Printf("query:\n %s\n", data)

	if game.Players[1] == nil && query.From.ID != game.Players[0].ID {
		log.Printf("%s registered as player 2", query.From.UserName)
		game.Players[1] = query.From
	}

	if game.Players[game.CurrentTurn%2].ID != query.From.ID {

		fmt.Printf("player: %s\n", game.Players[game.CurrentTurn%2].UserName)

		log.Println("not your turn registered")
		return tgbotapi.NewCallback(query.ID, "not your turn!!")
	}

	var gameId, row, col int
	fmt.Sscanf(query.Data, "%d_%d_%d", &gameId, &row, &col)

	if game.Grid[row-1][col-1] == Null {
		game.Grid[row-1][col-1] = gameSpace(game.CurrentTurn % 2)

		if game.checkForWin() {
			return tgbotapi.NewCallback(query.ID, "")
		}

		if game.checkForTie() {
			return tgbotapi.NewCallback(query.ID, "")
		}

		game.CurrentTurn++
	} else {
		return tgbotapi.NewCallback(query.ID, "space occupied")
	}

	return tgbotapi.NewCallback(query.ID, "")
}

func (game *TicTacToeGame) checkForTie() bool {
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if game.Grid[row][col] == Null {
				return false
			}
		}
	}

	return true
}

func (game *TicTacToeGame) checkForWin() bool {
	for row := 0; row < 3; row++ {
		if equals(game.Grid[row][0], game.Grid[row][1], game.Grid[row][2]) {
			return true
		}
	}

	for col := 0; col < 3; col++ {
		if equals(game.Grid[0][col], game.Grid[1][col], game.Grid[2][col]) {
			return true
		}
	}

	if equals(game.Grid[0][0], game.Grid[1][1], game.Grid[2][2]) {
		return true
	}

	if equals(game.Grid[0][2], game.Grid[1][1], game.Grid[2][0]) {
		return true
	}

	return false
}

func equals(vals ...gameSpace) bool {
	if len(vals) == 0 {
		return true
	}

	firstValue := vals[0]

	if firstValue == Null {
		return false
	}

	for _, val := range vals[1:] {
		if val != firstValue {
			return false
		}
	}

	return true
}
