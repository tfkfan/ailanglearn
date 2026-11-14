package voicechat

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strings"
)

// App bundles the HTTP dependencies.
type App struct {
	cfg   *Config
	tts   *TTS
	agent *Agent
	webFS fs.FS
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/languages", a.handleLanguages)
	mux.HandleFunc("/api/models", a.handleModels)
	mux.HandleFunc("/api/session/start", a.handleStart)
	mux.HandleFunc("/api/session/message", a.handleMessage)
	mux.HandleFunc("/api/session/end", a.handleEnd)
	mux.HandleFunc("/api/tts", a.handleTTS)

	sub, err := fs.Sub(a.webFS, "web")
	if err != nil {
		log.Fatalf("embed web: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))

	return withCORS(withLogging(mux))
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"defaultModel": a.cfg.LLMModel,
		"ttsProvider":  a.cfg.TTSProvider,
	})
}

func (a *App) handleLanguages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"languages": languages})
}

func (a *App) handleModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	list := make([]ModelInfo, len(modelRegistry))
	for i, m := range modelRegistry {
		m.Available = a.providerAvailable(m.Provider)
		list[i] = m
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": list, "default": a.cfg.LLMModel})
}

func (a *App) providerAvailable(p string) bool {
	switch p {
	case "deepseek":
		return a.cfg.DeepSeekKey != ""
	case "openai":
		return a.cfg.OpenAIKey != ""
	case "anthropic":
		return a.cfg.AnthropicKey != ""
	default:
		return true // ollama, mock
	}
}

func (a *App) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Language string `json:"language"`
		Model    string `json:"model"`
	}
	if err := readJSON(w, r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	sess, err := a.agent.Start(req.Language, req.Model)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sess.ID,
		"language":  sess.Language,
		"model":     sess.Model,
		"topic":     sess.Topic,
		"greeting":  sess.Opening,
	})
}

func (a *App) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
		Text      string `json:"text"`
	}
	if err := readJSON(w, r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		httpError(w, http.StatusBadRequest, "text is empty")
		return
	}
	reply, demo, err := a.agent.Send(req.SessionID, req.Text)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reply": reply, "demo": demo})
}

func (a *App) handleEnd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		SessionID string `json:"sessionId"`
	}
	if err := readJSON(w, r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	goodbye, err := a.agent.End(req.SessionID)
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"goodbye": goodbye})
}

func (a *App) handleTTS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Text string `json:"text"`
		Lang string `json:"lang"`
	}
	if err := readJSON(w, r, &req); err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		httpError(w, http.StatusBadRequest, "text is empty")
		return
	}
	lang := req.Lang
	if lang == "" {
		lang = "en"
	}
	data, mime, err := a.tts.Speak(req.Text, lang)
	if err != nil {
		httpError(w, http.StatusBadGateway, "tts failed: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}
