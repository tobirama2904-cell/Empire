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

const (
	GH_URL   = "https://azure.com"
	GROQ_URL = "https://groq.com"
)

func askAI(prompt, sys string) string {
	ghTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	groqKey := os.Getenv("GROQ_KEY")

	for _, token := range ghTokens {
		token = strings.TrimSpace(token)
		if token == "" { continue }
		for _, m := range []string{"gpt-4o", "meta-llama-3.1-70b"} {
			res := callAPI(GH_URL, token, m, sys, prompt)
			if res != "" { return res }
		}
	}
	if groqKey != "" {
		return callAPI(GROQ_URL, groqKey, "llama-3.1-70b-versatile", sys, prompt)
	}
	return "🚀 Ошибка связи с ИИ."
}

func callAPI(apiURL, key, model, sys, prompt string) string {
	payload := AIRequest{Model: model, Messages: []Message{{Role: "system", Content: sys}, {Role: "user", Content: prompt}}}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 { return "" }
	defer resp.Body.Close()
	var data struct {
		Choices []struct{ Message struct{ Content string } } `json:"choices"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Choices) > 0 { return data.Choices[0].Message.Content }
	return ""
}

// Запуск конкретного бота
func runBot(token, roleName, systemPrompt string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("[%s] Ошибка: %v", roleName, err)
		return
	}
	log.Printf("[%s] Запущен: %s", roleName, bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { continue }
		go func(m *tgbotapi.Message) {
			bot.Send(tgbotapi.NewChatAction(m.Chat.ID, "typing"))
			ans := askAI(m.Text, systemPrompt)
			msg := tgbotapi.NewMessage(m.Chat.ID, ans)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}(update.Message)
	}
}

func main() {
	tokenManager := os.Getenv("TOKEN_MANAGER")
	tokenRealizator := os.Getenv("TOKEN_REALIZATOR")

	if tokenManager == "" || tokenRealizator == "" {
		log.Panic("ОШИБКА: Нужно указать TOKEN_MANAGER и TOKEN_REALIZATOR в Render!")
	}

	// Запускаем Менеджера
	go runBot(tokenManager, "МЕНЕДЖЕР", "Ты — Менеджер Империи. Твоя цель: управлять пользователем, следить за задачами и отвечать в ЛС.")

	// Запускаем Реализатора (он не спит и ждет команд)
	runBot(tokenRealizator, "РЕАЛИЗАТОР", "Ты — Реализатор. Ты можешь стать любым приложением, сайтом или кодом. Твоя задача: воплощать идеи в жизнь.")
}
