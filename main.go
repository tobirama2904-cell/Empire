package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)


var (
	ideas []string
	mu    sync.Mutex
)

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "OK") })
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	botR, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_REALIZATOR"))
	botM, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_MANAGER"))
	adminID := os.Getenv("ADMIN_ID")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Запущено!")

	go func() {
		for update := range updatesR {
			if update.Message == nil { continue }
						if update.Message.Text == "/run" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Приложение:")
				// Исправленные названия функций для версии v5
				twaBtn := tgbotapi.InlineKeyboardButton{
					Text: "🚀 Старт",
					WebApp: &tgbotapi.WebAppInfo{URL: "https://js.org"},
				}
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(twaBtn),
				)
				botR.Send(msg)
			}
			
		}
	}()

	for update := range updatesM {
		if update.Message == nil { continue }
		m := update.Message
		if strings.HasPrefix(strings.ToLower(m.Text), "идея") {
			mu.Lock()
			ideas = append(ideas, fmt.Sprintf("@%s: %s", m.From.UserName, m.Text))
			mu.Unlock()
			botM.Send(tgbotapi.NewMessage(m.Chat.ID, "✅ Записал!"))
		}
		if fmt.Sprintf("%d", m.From.ID) == adminID && m.Text == "/top" {
			mu.Lock()
			res := "Топ-5:\n"
			for i, v := range ideas {
				if i >= 5 { break }
				res += fmt.Sprintf("%d. %s\n", i+1, v)
			}
			mu.Unlock()
			botM.Send(tgbotapi.NewMessage(m.Chat.ID, res))
		}
	}
}
