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

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Структуры для общения с нейросетями
type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Провайдеры ИИ (правильные адреса)
const (
	GH_URL   = "https://azure.com"
	GROQ_URL = "https://groq.com"
)

// Функция автоматического выбора живого ИИ
func askUltimateAI(prompt, sys string) string {
	// Собираем все ключи из настроек сервера
	ghTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	groqKey := os.Getenv("GROQ_KEY")

	// 1. Пробуем GitHub (перебираем все токены, если их много)
	for _, token := range ghTokens {
		token = strings.TrimSpace(token)
		if token == "" { continue }
		
		// Список бесплатных моделей на GitHub
		models := []string{"gpt-4o", "meta-llama-3.1-70b", "mistral-large-2407"}
		for _, m := range models {
			res := callAPI(GH_URL, token, m, sys, prompt)
			if res != "" { return res } // Если ответил — возвращаем результат
		}
	}

	// 2. Если GitHub не ответил, пробуем Groq (самый быстрый)
	if groqKey != "" {
		res := callAPI(GROQ_URL, groqKey, "llama-3.1-70b-versatile", sys, prompt)
		if res != "" { return res }
	}

	return "🚀 Все линии связи заняты. Попробуй через минуту!"
}

// Универсальный вызов API
func callAPI(apiURL, key, model, sys, prompt string) string {
	payload := AIRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: sys},
			{Role: "user", Content: prompt},
		},
	}
	
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	
	// Если ошибка (лимит, неверный ключ и т.д.), возвращаем пусто
	if err != nil || resp.StatusCode != http.StatusOK {
		return "" 
	}
	defer resp.Body.Close()

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || len(data.Choices) == 0 {
		return ""
	}
	return data.Choices[0].Message.Content
}

func main() {
	// Берем токен Телеграм из настроек
	botToken := os.Getenv("TELEGRAM_APITOKEN")
	if botToken == "" {
		log.Panic("Ошибка: Не указан TELEGRAM_APITOKEN")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Бот запущен под аккаунтом %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { continue }
		
		// Отвечаем в отдельном потоке (горутине), чтобы бот не тормозил
		go func(m *tgbotapi.Message) {
			// Отправляем статус "печатает..."
			bot.Send(tgbotapi.NewChatAction(m.Chat.ID, tgbotapi.ChatActionTyping))
			
			ans := askUltimateAI(m.Text, "Ты — мощный ИИ-помощник.")
			msg := tgbotapi.NewMessage(m.Chat.ID, ans)
			msg.ReplyToMessageID = m.MessageID // Ответ на конкретное сообщение
			bot.Send(msg)
		}(update.Message)
	}
}
