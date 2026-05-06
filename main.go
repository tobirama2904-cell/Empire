package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

const GROQ_URL = "https://groq.com"

// ФУНКЦИЯ ОХОТНИК: Мгновенно находит живой ключ из твоего списка
func hunterAsk(prompt, sys string) string {
	keys := strings.Split(os.Getenv("GROQ_KEYS"), ",")
	
	// Охотник тестирует ключи без задержек
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" { continue }

		// Список моделей от мощных к быстрым
		models := []string{"llama-3.3-70b-specdec", "llama-3.3-70b-versatile", "llama3-70b-8192"}
		
		for _, m := range models {
			res := callGroq(key, m, sys, prompt)
			if res != "" { return res }
		}
	}
	return "⚡️ Все каналы перегружены. Попробуй еще раз."
}

func callGroq(key, model, sys, prompt string) string {
	payload := AIRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: sys},
			{Role: "user", Content: prompt},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", GROQ_URL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	// Ультра-быстрый таймаут для охотника
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return ""
	}
	defer resp.Body.Close()

	var data struct {
		Choices []struct{ Message struct{ Content string } } `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Choices) > 0 {
		return data.Choices[0].Message.Content
	}
	return ""
}

func runBot(token, role, systemPrompt string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil { return }
	
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { continue }
		go func(m *tgbotapi.Message) {
			bot.Send(tgbotapi.NewChatAction(m.Chat.ID, "typing"))
			ans := hunterAsk(m.Text, systemPrompt)
			msg := tgbotapi.NewMessage(m.Chat.ID, ans)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}(update.Message)
	}
}

func main() {
	// Чтобы Render не убивал процесс, даем ему "фиктивный" старт
	go func() {
		http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	}()

	t1 := os.Getenv("TOKEN_MANAGER")
	t2 := os.Getenv("TOKEN_REALIZATOR")

	go runBot(t1, "Manager", "Ты Менеджер. Управляй системой.")
	runBot(t2, "Realizator", "Ты Реализатор. Твори код и проекты.")
}
