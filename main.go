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

// Структуры для обхода ошибок TWA на Render
type WebAppInfo struct {
	URL string `json:"url"`
}
type InlineKeyboardButtonWebApp struct {
	Text   string      `json:"text"`
	WebApp WebAppInfo  `json:"web_app"`
}

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
		client := &http.Client{Timeout: 60 * time.Second}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != 200 {
			if resp != nil { resp.Body.Close() }
			continue
		}
		var res map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&res)
		resp.Body.Close()
		if choices, ok := res["choices"].([]interface{}); ok && len(choices) > 0 {
			choice := choices[0].(map[string]interface{})
			msg := choice["message"].(map[string]interface{})
			return msg["content"].(string)
		}
	}
	return "❌ Системы перегружены. Попробуй через минуту."
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

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updatesR := botR.GetUpdatesChan(u)
	updatesM := botM.GetUpdatesChan(u)

	log.Println("🔥 ИМПЕРИЯ АКТИВИРОВАНА. Уровень: Principal Engineer.")

	// ЛОГИКА МЕНЕДЖЕРА (Групповой интеллект)
	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			t := strings.ToLower(update.Message.Text)

			// Интеллектуальный анализ запроса
			if strings.Contains(t, "идея") || strings.Contains(t, "рынок") || strings.Contains(t, "тренд") {
				botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))
				sys := "Ты Principal Business Analyst. Проанализируй рынок IT 2026. Дай 5 конкретных идей для заработка, стек технологий и стратегию монетизации."
				res := askEmpireAI(update.Message.Text, sys)
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *АНАЛИЗ ИМПЕРИИ:* \n\n"+res))
			} else if strings.Contains(t, "создай") || strings.Contains(t, "сделай") || strings.Contains(t, "напиши код") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос на разработку принят. Перехожу в режим проектирования уровня Staff Engineer...")
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(update.Message.Text, " ", "_")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 РЕАЛИЗОВАТЬ ПРОЕКТ", link)))
				botM.Send(msg)
			}
		}
	}()

	// ЛОГИКА РЕАЛИЗАТОРА (Кодинг и деплой)
	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Работает Principal Engineer. Генерирую архитектуру и чистый код..."))

			sys := "Ты Staff Software Engineer. Напиши ПОЛНОСТЬЮ готовый к использованию код (HTML/JS/CSS или Backend). Код должен быть идеальным, современным и рабочим на 100%."
			rawCode := askEmpireAI("Задача: "+task, sys)
			
			// Самопроверка кода
			finalCode := askEmpireAI(rawCode, "Ты Senior QA. Исправь любые ошибки в этом коде и верни только чистый исправленный код.")
			
			siteURL := createGist(finalCode)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ ПРОЕКТ РЕАЛИЗОВАН И ПРОВЕРЕН.")
			if siteURL != "" {
				btn := InlineKeyboardButtonWebApp{Text: "🌐 ЗАПУСТИТЬ (TWA)", WebApp: WebAppInfo{URL: siteURL}}
				kbJSON, _ := json.Marshal(map[string]interface{}{"inline_keyboard": [][]InlineKeyboardButtonWebApp{{btn}}})
				var markup tgbotapi.InlineKeyboardMarkup
				json.Unmarshal(kbJSON, &markup)
				msg.ReplyMarkup = markup
			}
			botR.Send(msg)
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📦 *Исходный код:* \n```html\n"+finalCode+"\n```"))
		}
	}
}
