package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
)

const (
	addr               = ":8088"
	sessionName        = "go_ws_session"
	defaultOllamaURL   = "http://localhost:11434"
	defaultOllamaModel = "qwen3.5:9b"
	aiUnavailable      = "AI助手：本地 AI 暂时不可用，请确认 Ollama 是否已启动。"
	maxMessageRunes    = 800
)

var usersFile = envOrDefault("USERS_FILE", "users.json")

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type message struct {
	Type        string    `json:"type"`
	Username    string    `json:"username"`
	Content     string    `json:"content"`
	StickerURL  string    `json:"stickerUrl,omitempty"`
	Time        string    `json:"time"`
	OnlineCount int       `json:"onlineCount"`
	History     []message `json:"history,omitempty"`
}

type userStore struct {
	mu    sync.Mutex
	Users map[string]string `json:"users"`
}

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]string
}

type hub struct {
	clients map[*websocket.Conn]string
	history []message
	mu      sync.Mutex
}

type ollamaClient struct {
	baseURL string
	model   string
	client  *http.Client
}

func newUserStore() *userStore {
	return &userStore{Users: make(map[string]string)}
}

func newSessionStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]string)}
}

func newHub() *hub {
	return &hub{clients: make(map[*websocket.Conn]string)}
}

func now() string {
	return time.Now().Format("15:04:05")
}

func (s *userStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(usersFile)
	if errors.Is(err, os.ErrNotExist) {
		return s.saveLocked()
	}
	if err != nil {
		return err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return s.saveLocked()
	}

	var file struct {
		Users map[string]string `json:"users"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return err
	}
	if file.Users == nil {
		file.Users = make(map[string]string)
	}
	s.Users = file.Users
	return nil
}

func (s *userStore) saveLocked() error {
	data, err := json.MarshalIndent(struct {
		Users map[string]string `json:"users"`
	}{Users: s.Users}, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(usersFile, data, 0600)
}

func (s *userStore) create(username, password string) error {
	username = strings.TrimSpace(username)
	if err := validateCredentials(username, password); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Users[username]; exists {
		return errors.New("user already exists")
	}

	s.Users[username] = string(hash)
	return s.saveLocked()
}

func (s *userStore) check(username, password string) bool {
	s.mu.Lock()
	hash := s.Users[strings.TrimSpace(username)]
	s.mu.Unlock()

	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *sessionStore) create(username string) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.sessions[token] = username
	s.mu.Unlock()
	return token, nil
}

func (s *sessionStore) username(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username, ok := s.sessions[token]
	return username, ok
}

func (s *sessionStore) delete(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *sessionStore) usernameFromRequest(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return s.username(cookie.Value)
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 8,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json failed: %v", err)
	}
}

func envOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func newOllamaClient() *ollamaClient {
	return &ollamaClient{
		baseURL: strings.TrimRight(envOrDefault("OLLAMA_URL", defaultOllamaURL), "/"),
		model:   envOrDefault("OLLAMA_MODEL", defaultOllamaModel),
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *ollamaClient) chat(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	payload := struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		Stream bool `json:"stream"`
	}{
		Model:  c.model,
		Stream: false,
		Messages: []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Error != "" {
		return "", errors.New(result.Error)
	}

	reply := strings.TrimSpace(result.Message.Content)
	if reply == "" {
		return "", errors.New("empty ollama response")
	}
	return reply, nil
}

func readCredentials(r *http.Request) (string, string, error) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			return "", "", err
		}
		return strings.TrimSpace(body.Username), body.Password, nil
	}

	if err := r.ParseForm(); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(r.FormValue("username")), r.FormValue("password"), nil
}

func validStickerURL(stickerURL string) bool {
	allowed := map[string]bool{
		"/stickers/sticker-1.svg": true,
		"/stickers/sticker-2.svg": true,
		"/stickers/sticker-3.svg": true,
		"/stickers/sticker-4.svg": true,
		"/stickers/sticker-5.svg": true,
		"/stickers/sticker-6.svg": true,
	}
	return allowed[stickerURL]
}

func sanitizeType(messageType string) string {
	switch messageType {
	case "emoji", "sticker":
		return messageType
	default:
		return "chat"
	}
}

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-]{3,20}$`)

func validateCredentials(username, password string) error {
	if !validUsername.MatchString(username) {
		return errors.New("username must be 3-20 characters, only letters, digits, hyphens, underscores")
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	return nil
}

func parseAIPrompt(content string) (string, bool) {
	lower := strings.ToLower(content)
	if strings.HasPrefix(lower, "@ai ") || strings.HasPrefix(lower, "/ai ") {
		return strings.TrimSpace(content[4:]), true
	}
	return "", false
}

func isSummaryCommand(content string) bool {
	return strings.EqualFold(strings.TrimSpace(content), "/summary")
}

func limitMessage(content string) string {
	runes := []rune(content)
	if len(runes) <= maxMessageRunes {
		return content
	}
	return string(runes[:maxMessageRunes])
}

func (h *hub) historySnapshot(onlineCount int) []message {
	h.mu.Lock()
	defer h.mu.Unlock()

	history := make([]message, len(h.history))
	copy(history, h.history)
	for i := range history {
		history[i].OnlineCount = onlineCount
	}
	return history
}

func (h *hub) add(conn *websocket.Conn, username string) int {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[conn] = username
	return len(h.clients)
}

func (h *hub) remove(conn *websocket.Conn) (string, int, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	username, ok := h.clients[conn]
	if ok {
		delete(h.clients, conn)
	}
	return username, len(h.clients), ok
}

func (h *hub) onlineCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return len(h.clients)
}

func (h *hub) remember(msg message) {
	if msg.Type == "chat" || msg.Type == "emoji" || msg.Type == "sticker" || msg.Type == "ai" {
		h.history = append(h.history, msg)
		if len(h.history) > 20 {
			h.history = h.history[len(h.history)-20:]
		}
	}
}

func (h *hub) textHistoryLines() []string {
	h.mu.Lock()
	defer h.mu.Unlock()

	lines := make([]string, 0, len(h.history))
	for _, item := range h.history {
		if item.Type != "chat" && item.Type != "emoji" && item.Type != "ai" {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s %s：%s", item.Time, item.Username, content))
	}
	return lines
}

func (h *hub) broadcastAI(content string) {
	h.broadcast(message{
		Type:        "ai",
		Username:    "AI助手",
		Content:     content,
		Time:        now(),
		OnlineCount: h.onlineCount(),
	})
}

func (h *hub) askAI(ai *ollamaClient, prompt string) {
	reply, err := ai.chat(prompt)
	if err != nil {
		log.Printf("ollama chat failed: %v", err)
		reply = aiUnavailable
	}
	h.broadcastAI(reply)
}

func (h *hub) broadcast(msg message) {
	h.mu.Lock()
	h.remember(msg)
	data, err := json.Marshal(msg)
	if err != nil {
		h.mu.Unlock()
		log.Printf("marshal failed: %v", err)
		return
	}

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("broadcast failed: %v", err)
			conn.Close()
			delete(h.clients, conn)
		}
	}
	h.mu.Unlock()
}

func (h *hub) sendHistory(conn *websocket.Conn, onlineCount int) {
	history := h.historySnapshot(onlineCount)
	if len(history) == 0 {
		return
	}

	if err := conn.WriteJSON(message{
		Type:        "history",
		Username:    "系统",
		Content:     "最近 20 条聊天记录",
		Time:        now(),
		OnlineCount: onlineCount,
		History:     history,
	}); err != nil {
		log.Printf("send history failed: %v", err)
	}
}

func (h *hub) broadcastOnline() {
	h.broadcast(message{
		Type:        "online",
		Username:    "系统",
		Content:     "在线人数更新",
		Time:        now(),
		OnlineCount: h.onlineCount(),
	})
}

func (h *hub) handleWebSocket(sessions *sessionStore, ai *ollamaClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, ok := sessions.usernameFromRequest(r)
		if !ok {
			http.Error(w, "login required", http.StatusUnauthorized)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("upgrade failed: %v", err)
			return
		}

		nextOnlineCount := h.onlineCount() + 1
		h.sendHistory(conn, nextOnlineCount)
		onlineCount := h.add(conn, username)
		h.broadcast(message{
			Type:        "system",
			Username:    "系统",
			Content:     username + " 加入聊天室",
			Time:        now(),
			OnlineCount: onlineCount,
		})
		h.broadcastOnline()

		defer func() {
			username, onlineCount, removed := h.remove(conn)
			conn.Close()
			if removed {
				h.broadcast(message{
					Type:        "system",
					Username:    "系统",
					Content:     username + " 离开聊天室",
					Time:        now(),
					OnlineCount: onlineCount,
				})
				h.broadcastOnline()
			}
		}()

		for {
			var incoming message
			if err := conn.ReadJSON(&incoming); err != nil {
				log.Printf("read failed: %v", err)
				return
			}

			messageType := sanitizeType(incoming.Type)
			content := limitMessage(strings.TrimSpace(incoming.Content))
			stickerURL := strings.TrimSpace(incoming.StickerURL)
			if messageType == "sticker" {
				if !validStickerURL(stickerURL) {
					continue
				}
				content = "发送了一个表情包"
			} else if content == "" {
				continue
			}

			if messageType != "sticker" {
				if prompt, ok := parseAIPrompt(content); ok {
					if prompt == "" {
						continue
					}
					go h.askAI(ai, fmt.Sprintf("%s 在聊天室中向你提问：%s\n请用中文简洁回答。", username, prompt))
					continue
				}

				if isSummaryCommand(content) {
					lines := h.textHistoryLines()
					if len(lines) == 0 {
						h.broadcastAI("AI助手：目前还没有可总结的文字聊天记录。")
						continue
					}
					prompt := "请用中文总结下面聊天室最近 20 条文字聊天记录，提炼主要话题和待办事项，保持简洁：\n" + strings.Join(lines, "\n")
					go h.askAI(ai, prompt)
					continue
				}
			}

			h.broadcast(message{
				Type:        messageType,
				Username:    username,
				Content:     content,
				StickerURL:  stickerURL,
				Time:        now(),
				OnlineCount: h.onlineCount(),
			})
		}
	}
}

func main() {
	users := newUserStore()
	if err := users.load(); err != nil {
		log.Fatalf("load users failed: %v", err)
	}

	sessions := newSessionStore()
	h := newHub()
	ai := newOllamaClient()

	http.Handle("/stickers/", http.StripPrefix("/stickers/", http.FileServer(http.Dir("public/stickers"))))
	http.HandleFunc("/style.css", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "public/style.css") })
	http.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "public/app.js") })
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sessions.usernameFromRequest(r); ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.ServeFile(w, r, "public/login.html")
	})
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := sessions.usernameFromRequest(r); ok {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		http.ServeFile(w, r, "public/register.html")
	})

	http.HandleFunc("/api/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username, password, err := readCredentials(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		if err := users.create(username, password); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"username": username})
	})

	http.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		username, password, err := readCredentials(r)
		if err != nil || !users.check(username, password) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid username or password"})
			return
		}
		token, err := sessions.create(username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create session failed"})
			return
		}
		setSessionCookie(w, token)
		writeJSON(w, http.StatusOK, map[string]string{"username": username})
	})

	http.HandleFunc("/api/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cookie, err := r.Cookie(sessionName); err == nil {
			sessions.delete(cookie.Value)
		}
		clearSessionCookie(w)
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	http.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		username, ok := sessions.usernameFromRequest(r)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "login required"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"username": username})
	})

	http.HandleFunc("/ws", h.handleWebSocket(sessions, ai))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if _, ok := sessions.usernameFromRequest(r); !ok {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		http.ServeFile(w, r, "public/index.html")
	})

	log.Println("Go WebSocket Chatroom Demo listening on http://localhost:8088")
	log.Fatal(http.ListenAndServe(addr, nil))
}
