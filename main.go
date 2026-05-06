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
	groqKey := os.Getenv("GROQ_KEY")
	ghTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")

	// 1. Пробуем Groq (приоритет на скорость)
	if groqKey != "" {
		log.Println("--- Пробую Groq ---")
		res := callAPI(GROQ_URL, groqKey, "llama-3.1-70b-versatile", sys, prompt)
		if res != "" { return res }
	}

	// 2. Пробуем GitHub (резерв)
	for _, token := range ghTokens {
		token = strings.TrimSpace(token)
		if token == "" { continue }
		log.Println("--- Пробую GitHub ---")
		res := callAPI(GH_URL, token, "gpt-4o", sys, prompt)
		if res != "" { return res }
	}

	return "❌ Не удалось получить ответ от ИИ. Проверь ключи в настройках!"
}

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

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Ошибка сети: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API Error: статус %d (проверь ключ)", resp.StatusCode)
		return ""
	}

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Ошибка декодирования: %v", err)
		return ""
	}

	if len(data.Choices) > 0 {
		return data.Choices[0].Message.Content
	}
	return ""
}

func runBot(token, roleName, systemPrompt string) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Printf("[%s] Ошибка авторизации: %v", roleName, err)
		return
	}
	log.Printf("[%s] Запущен: @%s", roleName, bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { continue }
		go func(m *tgbotapi.Message) {
			bot.Send(tgbotapi.NewChatAction(m.Chat.ID, "typing"))
			log.Printf("[%s] Сообщение от %s: %s", roleName, m.From.UserName, m.Text)
			
			ans := askAI(m.Text, systemPrompt)
			
			msg := tgbotapi.NewMessage(m.Chat.ID, ans)
			msg.ParseMode = "Markdown"
			bot.Send(msg)
		}(update.Message)
	}
}

func main() {
	// Ждем 5 секунд, чтобы Render не ругался на порты сразу
	time.Sleep(5 * time.Second)
	
	t1 := os.Getenv("TOKEN_MANAGER")
	t2 := os.Getenv("TOKEN_REALIZATOR")

	if t1 == "" || t2 == "" {
		log.Fatal("Критическая ошибка: TOKEN_MANAGER или TOKEN_REALIZATOR пусты!")
	}

	go runBot(t1, "МЕНЕДЖЕР", "Ты Менеджер. Следи за порядком.")
	runBot(t2, "РЕАЛИЗАТОР", "Ты Реализатор. Твори код и сайты.")
}
