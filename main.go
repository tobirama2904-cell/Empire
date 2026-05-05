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

// Хранилище идей
var (
	ideas []string
	mu    sync.Mutex
)

func main() {
	// 1. ПОРТ ДЛЯ RENDER (Обязательно!)
	// Render дает порт через переменную окружения PORT
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Бот запущен и работает!")
		})
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	// 2. ТОКЕНЫ И НАСТРОЙКИ
	tokenRealizator := os.Getenv("TOKEN_REALIZATOR")
	tokenManager := os.Getenv("TOKEN_MANAGER")
	adminID := os.Getenv("ADMIN_ID") // Берем как строку для сравнения

	botR, err := tgbotapi.NewBotAPI(tokenRealizator)
	if err != nil {
		log.Panic("Ошибка Реализатора:", err)
	}

	botM, err := tgbotapi.NewBotAPI(tokenManager)
	if err != nil {
		log.Panic("Ошибка Менеджера:", err)
	}

	// 3. ЗАПУСК ОБРАБОТКИ
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Боты успешно запущены на Render!")

	// Логика Реализатора (Бот 1)
	go func() {
		for update := range updatesR {
			if update.Message == nil { continue }
			msg := update.Message

			if msg.Text == "/run" {
				// Ссылка на TWA (можно заменить на свою)
				twaBtn := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonWebApp("🚀 Запустить Приложение", tgbotapi.WebAppInfo{URL: "https://js.org"}),
					),
				)
				reply := tgbotapi.NewMessage(msg.Chat.ID, "Твой код скомпилирован в мини-приложение:")
				reply.ReplyMarkup = twaBtn
				botR.Send(reply)
			} else {
				botR.Send(tgbotapi.NewMessage(msg.Chat.ID, "🤖 Я Реализатор. Пиши идею — сделаю код.\nКоманда /run запустит результат."))
			}
		}
	}()

	// Логика Менеджера (Бот 2)
	for update := range updatesM {
		if update.Message == nil { continue }
		msg := update.Message
		text := msg.Text
		strUserID := fmt.Sprintf("%d", msg.From.ID)

		// Работа в Группе: Сбор идей
		if msg.Chat.IsGroup() || msg.Chat.IsSuperGroup() {
			if strings.HasPrefix(strings.ToLower(text), "идея") {
				mu.Lock()
				ideas = append(ideas, fmt.Sprintf("@%s: %s", msg.From.UserName, text))
				mu.Unlock()
				
				res := tgbotapi.NewMessage(msg.Chat.ID, "💡 Крутая идея! Менеджер сохранил её для админа.")
				res.ReplyToMessageID = msg.MessageID
				botM.Send(res)
			}
		}

		// Работа в Личке: Топ-5 для Админа
		if msg.Chat.IsPrivate() && strUserID == adminID {
			if text == "/top" {
				mu.Lock()
				var report string
				if len(ideas) == 0 {
					report = "Список идей пока пуст."
				} else {
					report = "📊 Последние 5 идей из группы:\n\n"
					count := 0
					for i := len(ideas) - 1; i >= 0 && count < 5; i-- {
						report += fmt.Sprintf("%d. %s\n", count+1, ideas[i])
						count++
					}
				}
				mu.Unlock()
				botM.Send(tgbotapi.NewMessage(msg.Chat.ID, report))
			}
		}
	}
}
