package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	tgbotapi "://github.com"
)

var (
	ideas []string
	mu    sync.Mutex
)

func main() {
	// Обязательная часть для Render
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Сервис активен")
		})
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// Токены из настроек Render
	tokenR := os.Getenv("TOKEN_REALIZATOR")
	tokenM := os.Getenv("TOKEN_MANAGER")
	adminID := os.Getenv("ADMIN_ID")

	botR, _ := tgbotapi.NewBotAPI(tokenR)
	botM, _ := tgbotapi.NewBotAPI(tokenM)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Боты запущены!")

	// Бот-Реализатор
	go func() {
		for update := range updatesR {
			if update.Message == nil { continue }
			if update.Message.Text == "/run" {
				btn := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonWebApp("🚀 Запустить", tgbotapi.WebAppInfo{URL: "https://js.org"}),
					),
				)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Приложение готово:")
				msg.ReplyMarkup = btn
				botR.Send(msg)
			}
		}
	}()

	// Бот-Менеджер
	for update := range updatesM {
		if update.Message == nil { continue }
		msg := update.Message
		
		if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
			if strings.HasPrefix(strings.ToLower(msg.Text), "идея") {
				mu.Lock()
				ideas = append(ideas, fmt.Sprintf("@%s: %s", msg.From.UserName, msg.Text))
				mu.Unlock()
				botM.Send(tgbotapi.NewMessage(msg.Chat.ID, "💡 Идея сохранена!"))
			}
		}

		if fmt.Sprintf("%d", msg.From.ID) == adminID && msg.Text == "/top" {
			mu.Lock()
			res := "📊 Топ-5 идей:\n"
			for i, v := range ideas {
				if i >= 5 { break }
				res += fmt.Sprintf("%d. %s\n", i+1, v)
			}
			mu.Unlock()
			botM.Send(tgbotapi.NewMessage(msg.Chat.ID, res))
		}
	}
}
