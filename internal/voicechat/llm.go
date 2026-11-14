package voicechat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is a single chat turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAICompat struct {
	base string
	key  string
}

type anthropicClient struct {
	base string
	key  string
}

// LLM dispatches chat requests to the right provider for a given model id.
type LLM struct {
	http      *http.Client
	deepseek  openAICompat
	openai    openAICompat
	ollama    openAICompat
	anthropic anthropicClient
}

func newLLM(cfg *Config) *LLM {
	return &LLM{
		http:      &http.Client{Timeout: 90 * time.Second},
		deepseek:  openAICompat{base: cfg.DeepSeekBase, key: cfg.DeepSeekKey},
		openai:    openAICompat{base: cfg.OpenAIBase, key: cfg.OpenAIKey},
		ollama:    openAICompat{base: cfg.OllamaBase, key: ""},
		anthropic: anthropicClient{base: cfg.AnthropicBase, key: cfg.AnthropicKey},
	}
}

// Chat sends the conversation to the provider for the given model. When the
// provider has no credentials, it degrades to the offline demo and reports
// demo=true so the caller can label the response accordingly.
func (l *LLM) Chat(model, lang, system string, history []Message) (reply string, demo bool, err error) {
	switch providerForModel(model) {
	case "openai":
		if l.openai.key == "" {
			return mockReply(lang, history), true, nil
		}
		s, err := l.openai.chat(l.http, model, system, history)
		return s, false, err
	case "anthropic":
		if l.anthropic.key == "" {
			return mockReply(lang, history), true, nil
		}
		s, err := l.anthropic.chat(l.http, model, system, history)
		return s, false, err
	case "ollama":
		s, err := l.ollama.chat(l.http, model, system, history)
		return s, false, err
	case "mock":
		return mockReply(lang, history), true, nil
	default: // deepseek
		if l.deepseek.key == "" {
			return mockReply(lang, history), true, nil
		}
		s, err := l.deepseek.chat(l.http, model, system, history)
		return s, false, err
	}
}

func (c openAICompat) chat(hc *http.Client, model, system string, history []Message) (string, error) {
	msgs := make([]Message, 0, len(history)+1)
	msgs = append(msgs, Message{Role: "system", Content: system})
	msgs = append(msgs, history...)

	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"messages":    msgs,
		"temperature": 0.8,
	})

	url := strings.TrimRight(c.base, "/") + "/chat/completions"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Set("Authorization", "Bearer "+c.key)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream %d: %s", resp.StatusCode, truncate(string(data), 300))
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", errors.New("upstream returned no choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

func (c anthropicClient) chat(hc *http.Client, model, system string, history []Message) (string, error) {
	msgs := make([]map[string]string, 0, len(history))
	for _, m := range history {
		role := m.Role
		if role == "system" {
			role = "user"
		}
		msgs = append(msgs, map[string]string{"role": role, "content": m.Content})
	}

	body, _ := json.Marshal(map[string]any{
		"model":       model,
		"max_tokens":  1024,
		"system":      system,
		"messages":    msgs,
		"temperature": 0.8,
	})

	url := strings.TrimRight(c.base, "/") + "/v1/messages"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.key)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream %d: %s", resp.StatusCode, truncate(string(data), 300))
	}

	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	var parts []string
	for _, p := range out.Content {
		if p.Type == "text" {
			parts = append(parts, p.Text)
		}
	}
	if len(parts) == 0 {
		return "", errors.New("upstream returned no text")
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}

var mockFollowups = map[string][]string{
	"en": {
		"That's interesting — tell me a little more about that.",
		"Nice! And why do you think that is?",
		"I see. Could you give me an example?",
		"Great. What happened next?",
	},
	"es": {
		"¡Qué interesante! Cuéntame un poco más sobre eso.",
		"¡Muy bien! ¿Y por qué crees que es así?",
		"Ya veo. ¿Puedes darme un ejemplo?",
		"¡Buena respuesta! ¿Qué pasó después?",
	},
	"fr": {
		"C'est intéressant ! Parle-moi un peu plus de ça.",
		"Super ! Et pourquoi penses-tu cela ?",
		"Je vois. Peux-tu me donner un exemple ?",
		"Bonne réponse ! Que s'est-il passé ensuite ?",
	},
}

func mockReply(lang string, history []Message) string {
	list := mockFollowups[lang]
	if len(list) == 0 {
		list = mockFollowups["en"]
	}
	n := 0
	for _, m := range history {
		if m.Role == "assistant" {
			n++
		}
	}
	return list[n%len(list)]
}
