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

// Структуры для API
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

// Функция выбора живого ИИ (Автоматизация смены ключей)
func askUltimateAI(prompt, sys string) string {
	ghTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	groqKey := os.Getenv("GROQ_KEY")

	// 1. Пробуем GitHub (перебор всех твоих токенов)
	for _, token := range ghTokens {
		token = strings.TrimSpace(token)
		if token == "" { continue }
		
		models := []string{"gpt-4o", "meta-llama-3.1-70b", "mistral-large-2407"}
		for _, m := range models {
			res := callAPI(GH_URL, token, m, sys, prompt)
			if res != "" { return res }
		}
	}

	// 2. Резервный Groq (если GitHub не алё)
	if groqKey != "" {
		res := callAPI(GROQ_URL, groqKey, "llama-3.1-70b-versatile", sys, prompt)
		if res != "" { return res }
	}

	return "🚀 *Все ИИ линии перегружены.* Попробуй через минуту!"
}

// Запрос к нейросети
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
	botToken := os.Getenv("TELEGRAM_APITOKEN")
	if botToken == "" {
		log.Panic("ОШИБКА: Забудь запустить переменную TELEGRAM_APITOKEN!")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Бот успешно запущен: %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { continue }
		
		go func(m *tgbotapi.Message) {
			// Показываем, что бот "думает"
			bot.Send(tgbotapi.NewChatAction(m.Chat.ID, "typing"))
			
			log.Printf("[%s] спросил: %s", m.From.UserName, m.Text)
			
			ans := askUltimateAI(m.Text, "Ты — мощный ИИ-помощник. Отвечай четко.")
			
			msg := tgbotapi.NewMessage(m.Chat.ID, ans)
			msg.ReplyToMessageID = m.MessageID
			msg.ParseMode = "Markdown" // Делает текст красивым
			
			bot.Send(msg)
		}(update.Message)
	}
}
