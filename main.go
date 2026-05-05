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

// Очередь "Империя": лучшие модели от мощных к быстрым
var modelQueue = []string{
	"gpt-4o",              // Интеллект Principal Engineer
	"claude-3-5-sonnet",   // Идеальный код
	"meta-llama-3.1-405b", // Аналитика рынка
	"meta-llama-3.1-70b",  // Скорость
	"gpt-4o-mini",         // Резерв
}

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func askEmpireAI(prompt string, sysPrompt string) string {
	token := os.Getenv("GITHUB_TOKEN")
	url := "https://azure.com"

	// Если задача мелкая, экономим мощные модели
	startIndex := 0
	if len(prompt) < 150 && !strings.Contains(strings.ToLower(prompt), "код") {
		startIndex = 3 
	}

	// Цикл ротации: если модель выдает ошибку, берем следующую МГНОВЕННО
	for i := startIndex; i < len(modelQueue); i++ {
		currentModel := modelQueue[i]
		log.Printf("[ИМПЕРИЯ] Запрос к модели: %s", currentModel)

		reqBody := AIRequest{
			Model: currentModel,
			Messages: []Message{
				{Role: "system", Content: sysPrompt},
				{Role: "user", Content: prompt},
			},
		}

		jsonData, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			continue // Ошибка сети -> следующая модель
		}

		if resp.StatusCode != 200 {
			log.Printf("[ЛИМИТ] %s недоступна, меняю ИИ...", currentModel)
			resp.Body.Close()
			continue // Лимит исчерпан -> следующая модель моментально
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if choices, ok := result["choices"].([]interface{}); ok && len(choices) > 0 {
			return choices.(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
		}
	}
	return "❌ Все ресурсы Империи на сегодня исчерпаны. Лимиты GitHub обновятся через 24 часа."
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	go http.ListenAndServe(":"+port, nil)

	botR, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_REALIZATOR"))
	botM, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_MANAGER"))
	adminID := os.Getenv("ADMIN_ID")

	u := tgbotapi.NewUpdate(0)
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Империя запущена на GitHub Models!")

	// МЕНЕДЖЕР: Группы и Анализ
	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			text := update.Message.Text
			
			if strings.Contains(strings.ToLower(text), "идея") {
				botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
				sys := "Ты Principal Analyst. Дай 5 идей для заработка в IT (тренды 2026). Будь краток и конкретен."
				res := askEmpireAI(text, sys)
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *Анализ:* \n\n"+res))
			}

			if strings.Contains(strings.ToLower(text), "создай") || strings.Contains(strings.ToLower(text), "сделай") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос принят. Нажми для реализации уровня Principal.")
				btn := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonURL("🚀 Реализовать", "https://t.me"+botR.Self.UserName+"?start=build_"+strings.ReplaceAll(text, " ", "_")),
					),
				)
				msg.ReplyMarkup = btn
				botM.Send(msg)
			}
		}
	}()

	// РЕАЛИЗАТОР: Код и TWA
	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start build_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start build_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Работает Principal Engineer. Проектирую..."))
			
			sys := "Ты Principal Software Engineer. Выдай полный, рабочий, идеальный код. Стек выбирай сам под задачу."
			code := askEmpireAI(task, sys)
			
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Проект готов:\n\n"+code))
		}
	}
}
