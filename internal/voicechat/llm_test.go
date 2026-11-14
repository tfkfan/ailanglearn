package voicechat

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockReply(t *testing.T) {
	for _, lang := range []string{"en", "es", "fr"} {
		if got := mockReply(lang, nil); got == "" {
			t.Fatalf("mockReply(%q) is empty", lang)
		}
	}
	if got := mockReply("xx", nil); got == "" {
		t.Fatal("mockReply(xx) is empty; expected English fallback")
	}

	first := mockReply("en", nil)
	second := mockReply("en", []Message{{Role: "assistant", Content: "x"}})
	if first == second {
		t.Fatalf("expected rotation, got same reply %q", first)
	}
}

func TestChatFallsBackToDemoWithoutKey(t *testing.T) {
	llm := newLLM(&Config{})
	for _, model := range []string{
		"deepseek-v4-pro", "deepseek-chat", "gpt-4o-mini", "gpt-4o",
		"claude-3-5-haiku-latest", "claude-3-7-sonnet-latest", "mock-tutor",
	} {
		reply, demo, err := llm.Chat(model, "en", "system prompt", nil)
		if err != nil {
			t.Fatalf("Chat(%q) err = %v", model, err)
		}
		if !demo {
			t.Fatalf("Chat(%q) demo = false, want true", model)
		}
		if reply == "" {
			t.Fatalf("Chat(%q) reply is empty", model)
		}
	}
}

func TestNewLLM(t *testing.T) {
	cfg := &Config{
		DeepSeekBase:  "http://ds",
		DeepSeekKey:   "k1",
		OpenAIBase:    "http://oa",
		OpenAIKey:     "k2",
		AnthropicBase: "http://an",
		AnthropicKey:  "k3",
		OllamaBase:    "http://ol",
	}
	llm := newLLM(cfg)
	if llm.http == nil {
		t.Fatal("http client is nil")
	}
	if llm.deepseek.base != "http://ds" || llm.deepseek.key != "k1" {
		t.Fatalf("deepseek = %+v", llm.deepseek)
	}
	if llm.openai.base != "http://oa" || llm.openai.key != "k2" {
		t.Fatalf("openai = %+v", llm.openai)
	}
	if llm.anthropic.base != "http://an" || llm.anthropic.key != "k3" {
		t.Fatalf("anthropic = %+v", llm.anthropic)
	}
	if llm.ollama.base != "http://ol" {
		t.Fatalf("ollama = %+v", llm.ollama)
	}
}

func TestOpenAICompatChat(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"hello there"}}]}`)
	}))
	defer srv.Close()

	c := openAICompat{base: srv.URL, key: "secret"}
	reply, err := c.chat(srv.Client(), "m", "sys", []Message{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "hello there" {
		t.Fatalf("reply = %q", reply)
	}
	if gotPath != "/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer secret" {
		t.Fatalf("auth = %q", gotAuth)
	}
}

func TestOpenAICompatChatError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()

	c := openAICompat{base: srv.URL}
	if _, err := c.chat(srv.Client(), "m", "", nil); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestAnthropicChat(t *testing.T) {
	var gotPath, gotAuth, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content":[{"type":"text","text":"bonjour"}]}`)
	}))
	defer srv.Close()

	c := anthropicClient{base: srv.URL, key: "ak"}
	reply, err := c.chat(srv.Client(), "m", "sys", []Message{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "yo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "bonjour" {
		t.Fatalf("reply = %q", reply)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "ak" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotVersion != "2023-06-01" {
		t.Fatalf("version = %q", gotVersion)
	}
}
