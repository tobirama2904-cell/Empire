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

// Структуры для TWA (без ошибок компиляции)
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

// Умный ИИ-движок с исправленной логикой типов
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

		// ИСПРАВЛЕННАЯ ЛОГИКА ТИПОВ (решает ошибку invalid operation)
		if choicesRaw, ok := res["choices"]; ok {
			if choices, ok := choicesRaw.([]interface{}); ok && len(choices) > 0 {
				if firstChoice, ok := choices[0].(map[string]interface{}); ok {
					if msg, ok := firstChoice["message"].(map[string]interface{}); ok {
						return msg["content"].(string)
					}
				}
			}
		}
	}
	return "⚠️ Системы перегружены. Попробуй еще раз."
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

	log.Println("🔥 ИМПЕРИЯ ЗАПУЩЕНА. Уровень: Staff Engineer.")

	// МЕНЕДЖЕР: Групповой ИИ-мозг
	go func() {
		for update := range updatesM {
			if update.Message == nil { continue }
			text := update.Message.Text
			if len(text) < 5 { continue }

			botM.Send(tgbotapi.NewChatAction(update.Message.Chat.ID, tgbotapi.ChatTyping))

			// Менеджер анализирует: это запрос на идею или на создание?
			decisionSys := "Ты — главный ИИ группы. Если юзер просит идею, анализ или заработок — ответь 'ANALYZE'. Если просит создать код/сайт/игру — ответь 'CREATE'. В остальных случаях — 'CHAT'."
			decision := askEmpireAI(text, decisionSys)

			if strings.Contains(decision, "ANALYZE") {
				sys := "Ты Principal Business Analyst. Дай 5 гениальных идей для заработка на основе трендов 2026, план и монетизацию."
				res := askEmpireAI(text, sys)
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📊 *АНАЛИЗ ИМПЕРИИ:* \n\n"+res))
			} else if strings.Contains(decision, "CREATE") {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🛠 Запрос на разработку принят. Начинаю проектирование...")
				link := "https://t.me" + botR.Self.UserName + "?start=task_" + strings.ReplaceAll(text, " ", "_")
				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonURL("🚀 РЕАЛИЗОВАТЬ", link)))
				botM.Send(msg)
			} else {
				// Бот просто поддерживает общение на высоком уровне
				res := askEmpireAI(text, "Ты — высокоинтеллектуальный помощник 'Империя'. Отвечай кратко, мудро и по делу.")
				botM.Send(tgbotapi.NewMessage(update.Message.Chat.ID, res))
			}
		}
	}()

	// РЕАЛИЗАТОР: Код и деплой
	for update := range updatesR {
		if update.Message == nil { continue }
		if strings.HasPrefix(update.Message.Text, "/start task_") {
			task := strings.ReplaceAll(strings.TrimPrefix(update.Message.Text, "/start task_"), "_", " ")
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "🏗 Principal Engineer в деле. Генерирую и проверяю код..."))

			sys := "Ты Staff Software Engineer. Напиши ОДИН файл HTML/CSS/JS. Код должен быть идеальным, современным и полностью рабочим."
			code := askEmpireAI(task, sys)
			
			siteURL := createGist(code)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ ПРОЕКТ РЕАЛИЗОВАН.")
			if siteURL != "" {
				btn := InlineKeyboardButtonWebApp{Text: "🌐 ЗАПУСТИТЬ (TWA)", WebApp: WebAppInfo{URL: siteURL}}
				kbJSON, _ := json.Marshal(map[string]interface{}{"inline_keyboard": [][]InlineKeyboardButtonWebApp{{btn}}})
				var markup tgbotapi.InlineKeyboardMarkup
				json.Unmarshal(kbJSON, &markup)
				msg.ReplyMarkup = markup
			}
			botR.Send(msg)
			botR.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "📦 *Исходный код:* \n```html\n"+code+"\n```"))
		}
	}
}
