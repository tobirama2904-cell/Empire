package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Модели для ротации
var modelQueue = []string{
	"gpt-4o",
	"claude-3-5-sonnet",
	"meta-llama-3.1-405b",
	"gpt-4o-mini",
}

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Универсальная функция запроса к AI (GitHub Models)
func askAI(prompt, sysPrompt string) string {
	token := os.Getenv("GITHUB_TOKEN")
	url := "https://azure.com"

	for _, model := range modelQueue {
		reqBody := AIRequest{
			Model: model,
			Messages: []Message{
				{Role: "system", Content: sysPrompt},
				{Role: "user", Content: prompt},
			},
		}
		jsonData, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 45 * time.Second}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil { resp.Body.Close() }
			continue // Ротация: если лимит или ошибка, берем следующую модель
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
			return choices.(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
		}
	}
	return "❌ Все ресурсы сейчас недоступны. Попробуй позже."
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	go http.ListenAndServe(":"+port, nil)

	botR, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_REALIZATOR"))
	botM, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_MANAGER"))

	u := tgbotapi.NewUpdate(0)
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Система 'Империя' запущена 24/7!")

	// ЛОГИКА МЕНЕДЖЕРА
	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			text := strings.ToLower(update.Message.Text)

			if strings.Contains(text, "идея") {
				botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
				sys := "Ты Principal Business Analyst. Проанализируй тренды 2026 и дай 5 идей для заработка в IT."
				res := askAI(update.Message.Text, sys)
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *Анализ рынка:* \n\n"+res))
			}

			if strings.Contains(text, "создай") || strings.Contains(text, "сделай") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос принят. Нажми для реализации.")
				// Ссылка с параметром для Реализатора
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(update.Message.Text, " ", "_")
				btn := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 Реализовать проект", link)),
				)
				msg.ReplyMarkup = btn
				botM.Send(msg)
			}
		}
	}()

	// ЛОГИКА РЕАЛИЗАТОРА
	for update := range updatesR {
		if update.Message == nil { continue }

		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Работает Principal Engineer. Проектирую..."))

			sys := "Ты Staff Software Engineer. Выдай ПОЛНОСТЬЮ рабочий код (HTML/JS/CSS или Go/Python). Если это UI — делай современно."
			code := askAI(task, sys)

			// Отправляем код и кнопку TWA
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Проект готов:\n\n"+code)
			
			// Настройка TWA кнопки (исправлено под v5)
			twaBtn := tgbotapi.InlineKeyboardButton{
				Text: "🌐 Запустить Приложение (TWA)",
				WebApp: &tgbotapi.WebAppInfo{URL: "https://js.org"}, // Сюда можно вставить хостинг твоего кода
			}
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(twaBtn),
			)
			botR.Send(msg)
		}
	}
}
