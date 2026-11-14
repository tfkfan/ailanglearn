package voicechat

import (
	"log"
	"os"
)

// Config holds all runtime configuration, populated from environment variables.
type Config struct {
	Port     string
	LLMModel string

	DeepSeekBase  string
	DeepSeekKey   string
	OpenAIBase    string
	OpenAIKey     string
	AnthropicBase string
	AnthropicKey  string
	OllamaBase    string

	TTSProvider string
	TTSBase     string
	TTSKey      string
	TTSModel    string
	TTSVoice    string
}

func loadConfig() *Config {
	return &Config{
		Port:          getenv("PORT", "8080"),
		LLMModel:      getenv("LLM_MODEL", "deepseek-v4-pro"),
		DeepSeekBase:  getenv("DEEPSEEK_API_BASE", "https://api.deepseek.com"),
		DeepSeekKey:   getenv("DEEPSEEK_API_KEY", ""),
		OpenAIBase:    getenv("OPENAI_API_BASE", "https://api.openai.com/v1"),
		OpenAIKey:     getenv("OPENAI_API_KEY", ""),
		AnthropicBase: getenv("ANTHROPIC_API_BASE", "https://api.anthropic.com"),
		AnthropicKey:  getenv("ANTHROPIC_API_KEY", ""),
		OllamaBase:    getenv("OLLAMA_BASE", "http://localhost:11434"),
		TTSProvider:   getenv("TTS_PROVIDER", "google"),
		TTSBase:       getenv("TTS_API_BASE", "https://api.openai.com/v1"),
		TTSKey:        getenv("TTS_API_KEY", ""),
		TTSModel:      getenv("TTS_MODEL", "tts-1"),
		TTSVoice:      getenv("TTS_VOICE", "alloy"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Lang describes one supported learning language.
type Lang struct {
	Code   string `json:"code"`   // ISO code used internally
	Name   string `json:"name"`   // English name (used in prompts)
	Native string `json:"native"` // native name (used in the UI)
	TTS    string `json:"tts"`    // language code for server TTS
	STT    string `json:"stt"`    // BCP-47 code for browser speech recognition
}

var languages = []Lang{
	{Code: "en", Name: "English", Native: "English", TTS: "en", STT: "en-US"},
	{Code: "es", Name: "Spanish", Native: "Español", TTS: "es", STT: "es-ES"},
	{Code: "fr", Name: "French", Native: "Français", TTS: "fr", STT: "fr-FR"},
}

func langByCode(code string) (Lang, bool) {
	for _, l := range languages {
		if l.Code == code {
			return l, true
		}
	}
	return Lang{}, false
}

// ModelInfo describes a selectable agent model.
type ModelInfo struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Provider  string `json:"provider"`
	Available bool   `json:"available"`
}

var modelRegistry = []ModelInfo{
	{ID: "deepseek-v4-pro", Label: "DeepSeek V4 Pro", Provider: "deepseek"},
	{ID: "deepseek-chat", Label: "DeepSeek Chat", Provider: "deepseek"},
	{ID: "gpt-4o-mini", Label: "GPT-4o mini", Provider: "openai"},
	{ID: "gpt-4o", Label: "GPT-4o", Provider: "openai"},
	{ID: "claude-3-5-haiku-latest", Label: "Claude 3.5 Haiku", Provider: "anthropic"},
	{ID: "claude-3-7-sonnet-latest", Label: "Claude 3.7 Sonnet", Provider: "anthropic"},
	{ID: "llama3.2", Label: "Llama 3.2 · local", Provider: "ollama"},
	{ID: "mock-tutor", Label: "Demo tutor · offline", Provider: "mock"},
}

func providerForModel(id string) string {
	for _, m := range modelRegistry {
		if m.ID == id {
			return m.Provider
		}
	}
	return "deepseek"
}

func logConfig(cfg *Config) {
	log.Printf("default model: %s", cfg.LLMModel)
	providers := []string{"mock"}
	if cfg.DeepSeekKey != "" {
		providers = append(providers, "deepseek")
	}
	if cfg.OpenAIKey != "" {
		providers = append(providers, "openai")
	}
	if cfg.AnthropicKey != "" {
		providers = append(providers, "anthropic")
	}
	providers = append(providers, "ollama")
	log.Printf("configured providers: %v", providers)
	log.Printf("tts provider: %s", cfg.TTSProvider)
	if cfg.DeepSeekKey == "" && cfg.OpenAIKey == "" && cfg.AnthropicKey == "" {
		log.Printf("note: no LLM API key set — running in offline demo mode (mock-tutor)")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
