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

var modelQueue = []string{"gpt-4o", "claude-3-5-sonnet", "meta-llama-3.1-405b", "gpt-4o-mini"}

type AIRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func askEmpireAI(prompt, sys string) string {
	token := os.Getenv("GITHUB_TOKEN")
	url := "https://azure.com"
	for _, m := range modelQueue {
		body, _ := json.Marshal(AIRequest{Model: m, Messages: []Message{{Role: "system", Content: sys}, {Role: "user", Content: prompt}}})
		req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := (&http.Client{Timeout: 45 * time.Second}).Do(req)
		if err != nil || resp.StatusCode != 200 { continue }
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		
		if choices, ok := res["choices"].([]interface{}); ok && len(choices) > 0 {
			choice := choices[0].(map[string]interface{})
			msg := choice["message"].(map[string]interface{})
			return msg["content"].(string)
		}
	}
	return ""
}

func createGist(code string) string {
	token := os.Getenv("GITHUB_TOKEN")
	url := "https://github.com"
	body := map[string]interface{}{"public": true, "files": map[string]interface{}{"index.html": map[string]string{"content": code}}}
	jsonB, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonB))
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
	adminID := os.Getenv("ADMIN_ID")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("Империя готова к запуску!")

	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			text := strings.ToLower(update.Message.Text)
			if strings.Contains(text, "идея") {
				res := askEmpireAI(text, "Ты Principal Analyst. Дай 5 идей для заработка в IT.")
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *Анализ:* \n\n"+res))
			}
			if strings.Contains(text, "создай") || strings.Contains(text, "сделай") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос принят.")
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(update.Message.Text, " ", "_")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 Реализовать", link)))
				botM.Send(msg)
			}
		}
	}()

	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Principal Engineer работает..."))
			
			rawCode := askEmpireAI(task, "Ты Principal Software Engineer. Напиши ОДИН файл HTML/CSS/JS. Только код.")
			siteURL := createGist(rawCode)
			
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ Проект готов!")
			if siteURL != "" {
				// Исправленный блок кнопки WebApp
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.InlineKeyboardButton{
							Text: "🌐 Открыть (TWA)",
							WebApp: &tgbotapi.WebAppInfo{URL: siteURL},
						},
					),
				)
			}
			botR.Send(msg)
		}
	}
}
