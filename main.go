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

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func askUltimateAI(prompt, sys string) string {
	ghTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	groqKey := os.Getenv("GROQ_KEY")
	hfToken := os.Getenv("HF_TOKEN")

	// 1. Пытаемся GitHub (мощный код)
	for _, token := range ghTokens {
		token = strings.TrimSpace(token)
		if token == "" { continue }
		models := []string{"gpt-4o", "claude-3-5-sonnet", "meta-llama-3.1-70b"}
		for _, m := range models {
			res := callAPI("https://azure.com", token, m, sys, prompt)
			if res != "" { return res }
		}
	}

	// 2. Пытаемся Groq (скорость)
	if groqKey != "" {
		res := callAPI("https://groq.com", groqKey, "llama-3.1-70b-versatile", sys, prompt)
		if res != "" { return res }
	}

	// 3. Резерв Hugging Face
	if hfToken != "" {
		res := callAPI("https://huggingface.co", hfToken, "mistral", sys, prompt)
		if res != "" { return res }
	}

	return "🚀 Все линии связи заняты. Империя расширяет лимиты... Попробуй через 60 секунд."
}

func callAPI(apiURL, key, model, sys, prompt string) string {
	body, _ := json.Marshal(AIRequest{Model: model, Messages: []Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: prompt},
	}})
	req, _ := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil { resp.Body.Close() }
		return ""
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	// ИСПРАВЛЕННЫЙ ВЫВОД (решает ошибку choices)
	if choices, ok := data["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := firstChoice["message"].(map[string]interface{}); ok {
				return msg["content"].(string)
			}
		}
	}
	return ""
}

func createGist(code string) string {
	tokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	token := strings.TrimSpace(tokens[0])
	body := map[string]interface{}{"public": true, "files": map[string]interface{}{"index.html": map[string]string{"content": code}}}
	jsonB, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "https://github.com", bytes.NewBuffer(jsonB))
	req.Header.Set("Authorization", "token "+token)
	resp, _ := (&http.Client{}).Do(req)
	if resp == nil { return "" }
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if raw, ok := res["html_url"].(string); ok {
		return "https://github.io?" + raw
	}
	return ""
}

func main() {
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	go http.ListenAndServe(":"+port, nil)

	botR, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_REALIZATOR"))
	botM, _ := tgbotapi.NewBotAPI(os.Getenv("TOKEN_MANAGER"))

	updatesR := botR.GetUpdatesChan(tgbotapi.NewUpdate(0))
	updatesM := botM.GetUpdatesChan(tgbotapi.NewUpdate(0))

	log.Println("⚡️ ИМПЕРИЯ: ТИТАН ЗАПУЩЕН")

	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			t := strings.ToLower(update.Message.Text)
			if strings.Contains(t, "идея") || strings.Contains(t, "рынок") {
				botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
				res := askUltimateAI(t, "Ты Principal Global Analyst. Дай 5 идей на миллион.")
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *АНАЛИЗ ИМПЕРИИ:* \n\n"+res))
			}
			if strings.Contains(t, "создай") || strings.Contains(t, "сделай") {
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(t, " ", "_")
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос принят. Жми кнопку!")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 РЕАЛИЗОВАТЬ", link)))
				botM.Send(msg)
			}
		}
	}()

	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Работает Principal Engineer..."))
			
			code := askUltimateAI(task, "Ты Principal Software Engineer. Напиши ОДИН файл HTML/CSS/JS.")
			siteURL := createGist(code)
			
			ad := "\n\n📢 *Создано в Empire! Наш канал: https://t.me/JarvisEmpireCode*"
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ ПРОЕКТ ГОТОВ!"+ad)
			msg.ParseMode = "Markdown"
			
			if siteURL != "" {
				btnJSON := fmt.Sprintf(`{"inline_keyboard":[[{"text":"🌐 ОТКРЫТЬ (TWA)","web_app":{"url":"%s"}}]]}`, siteURL)
				var markup tgbotapi.InlineKeyboardMarkup
				json.Unmarshal([]byte(btnJSON), &markup)
				msg.ReplyMarkup = markup
			}
			botR.Send(msg)
		}
	}
}
