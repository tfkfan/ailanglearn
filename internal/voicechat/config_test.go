package voicechat

import "testing"

func TestGetenv(t *testing.T) {
	t.Setenv("TEST_GETENV", "hello")
	if got := getenv("TEST_GETENV", "fallback"); got != "hello" {
		t.Fatalf("getenv = %q, want %q", got, "hello")
	}
	if got := getenv("TEST_GETENV_MISSING", "fallback"); got != "fallback" {
		t.Fatalf("getenv missing = %q, want fallback", got)
	}
	t.Setenv("TEST_GETENV_EMPTY", "")
	if got := getenv("TEST_GETENV_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("getenv empty = %q, want fallback", got)
	}
}

func TestGetenvInt(t *testing.T) {
	t.Setenv("TEST_INT", "42")
	if got := getenvInt("TEST_INT", 5); got != 42 {
		t.Fatalf("getenvInt = %d, want 42", got)
	}

	cases := map[string]string{
		"TEST_INT_EMPTY": "",
		"TEST_INT_BAD":   "abc",
		"TEST_INT_NEG":   "-3",
		"TEST_INT_ZERO":  "0",
	}
	for name, val := range cases {
		t.Setenv(name, val)
	}
	for _, name := range []string{"TEST_INT_EMPTY", "TEST_INT_BAD", "TEST_INT_NEG", "TEST_INT_ZERO"} {
		if got := getenvInt(name, 5); got != 5 {
			t.Fatalf("getenvInt(%s) = %d, want 5", name, got)
		}
	}
}

func TestLangByCode(t *testing.T) {
	for _, code := range []string{"en", "es", "fr"} {
		l, ok := langByCode(code)
		if !ok || l.Code != code {
			t.Fatalf("langByCode(%q) = %+v, %v; want code %q", code, l, ok, code)
		}
	}
	if _, ok := langByCode("de"); ok {
		t.Fatal("langByCode(de) should not be ok")
	}
}

func TestLanguages(t *testing.T) {
	if len(languages) != 3 {
		t.Fatalf("languages = %d, want 3", len(languages))
	}
	for _, l := range languages {
		if l.Code == "" || l.Name == "" || l.Native == "" || l.TTS == "" || l.STT == "" {
			t.Fatalf("language %+v has an empty field", l)
		}
	}
}

func TestProviderForModel(t *testing.T) {
	cases := map[string]string{
		"deepseek-v4-pro":          "deepseek",
		"deepseek-chat":            "deepseek",
		"gpt-4o-mini":              "openai",
		"gpt-4o":                   "openai",
		"claude-3-5-haiku-latest":  "anthropic",
		"claude-3-7-sonnet-latest": "anthropic",
		"llama3.2":                 "ollama",
		"mock-tutor":               "mock",
		"unknown-model":            "deepseek",
	}
	for model, want := range cases {
		if got := providerForModel(model); got != want {
			t.Fatalf("providerForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestModelRegistry(t *testing.T) {
	if len(modelRegistry) == 0 {
		t.Fatal("modelRegistry is empty")
	}
	seen := map[string]bool{}
	for _, m := range modelRegistry {
		if m.ID == "" || m.Label == "" || m.Provider == "" {
			t.Fatalf("model %+v has an empty field", m)
		}
		if seen[m.ID] {
			t.Fatalf("duplicate model id %q", m.ID)
		}
		seen[m.ID] = true
	}
}

func TestTruncate(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello", 3, "hel…"},
	}
	for _, c := range cases {
		if got := truncate(c.s, c.n); got != c.want {
			t.Fatalf("truncate(%q, %d) = %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("PORT", "9999")
	t.Setenv("LLM_MODEL", "mock-tutor")
	t.Setenv("STT_SILENCE_SECONDS", "7")
	t.Setenv("PROMPTS_DIR", "/tmp/prompts")

	cfg := loadConfig()
	if cfg.Port != "9999" {
		t.Fatalf("Port = %q, want 9999", cfg.Port)
	}
	if cfg.LLMModel != "mock-tutor" {
		t.Fatalf("LLMModel = %q, want mock-tutor", cfg.LLMModel)
	}
	if cfg.STTSilenceSeconds != 7 {
		t.Fatalf("STTSilenceSeconds = %d, want 7", cfg.STTSilenceSeconds)
	}
	if cfg.PromptsDir != "/tmp/prompts" {
		t.Fatalf("PromptsDir = %q, want /tmp/prompts", cfg.PromptsDir)
	}
}
