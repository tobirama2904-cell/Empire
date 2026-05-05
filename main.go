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

// Ультра-ротация: GitHub -> Groq -> HuggingFace
func askUltimateAI(prompt, sys string) string {
	// 1. ПРОБУЕМ GITHUB (Интеллект)
	githubTokens := strings.Split(os.Getenv("GITHUB_TOKENS"), ",")
	for _, token := range githubTokens {
		token = strings.TrimSpace(token)
		models := []string{"gpt-4o", "claude-3-5-sonnet", "meta-llama-3.1-70b"}
		for _, m := range models {
			res := callGenericAPI("https://azure.com", token, m, sys, prompt)
			if res != "" { return res }
		}
	}

	// 2. ПРОБУЕМ GROQ (Скорость)
	groqKey := os.Getenv("GROQ_KEY")
	if groqKey != "" {
		res := callGenericAPI("https://groq.com", groqKey, "llama-3.1-70b-versatile", sys, prompt)
		if res != "" { return res }
	}

	// 3. ПРОБУЕМ HUGGING FACE (Безлимит)
	hfToken := os.Getenv("HF_TOKEN")
	if hfToken != "" {
		res := callGenericAPI("https://huggingface.co", hfToken, "mistral", sys, prompt)
		if res != "" { return res }
	}

	return "❌ Все системы перегружены. Империя ушла на перезагрузку лимитов."
}

func callGenericAPI(url, key, model, sys, prompt string) string {
	body, _ := json.Marshal(AIRequest{Model: model, Messages: []Message{
		{Role: "system", Content: sys},
		{Role: "user", Content: prompt},
	}})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil { resp.Body.Close() }
		return ""
	}
	defer resp.Body.Close()
	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if choices, ok := res["choices"].([]interface{}); ok && len(choices) > 0 {
		return choices.(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	}
	return ""
}

func createGist(code string) string {
	token := strings.TrimSpace(strings.Split(os.Getenv("GITHUB_TOKENS"), ",")[0])
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

	u := tgbotapi.NewUpdate(0)
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("⚡️ ИМПЕРИЯ: МУЛЬТИ-ДВИЖОК ЗАПУЩЕН (GitHub + Groq + HF)")

	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			t := update.Message.Text
			if strings.Contains(strings.ToLower(t), "идея") {
				botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
				res := askUltimateAI(t, "Ты Principal Global Analyst. Дай 5 идей на миллион.")
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *АНАЛИЗ:* \n\n"+res))
			}
			if strings.Contains(strings.ToLower(t), "создай") || strings.Contains(strings.ToLower(t), "сделай") {
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(t, " ", "_")
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос принят.")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 РЕАЛИЗОВАТЬ", link)))
				botM.Send(msg)
			}
		}
	}()

	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Проектирую через мощнейший доступный ИИ..."))
			code := askUltimateAI(task, "Ты Principal Software Engineer. Напиши ОДИН файл HTML/CSS/JS.")
			siteURL := createGist(code)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ ПРОЕКТ ГОТОВ.")
			if siteURL != "" {
				btnJSON := fmt.Sprintf(`{"inline_keyboard":[[{"text":"🌐 ОТКРЫТЬ (TWA)","web_app":{"url":"%s"}}]]}`, siteURL)
				var markup tgbotapi.InlineKeyboardMarkup
				json.Unmarshal([]byte(btnJSON), &markup)
				msg.ReplyMarkup = markup
			}
			botR.Send(msg)
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📦 *КОД:* \n```html\n"+code+"\n```"))
		}
	}
}
