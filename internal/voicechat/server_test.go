package voicechat

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	cfg := &Config{LLMModel: "mock-tutor", STTSilenceSeconds: 5, TTSProvider: "mock"}

	fsys := fstest.MapFS{
		"tutor_system.txt": {Data: []byte("tutor {{Language}}")},
		"topic_system.txt": {Data: []byte("json")},
		"topic_user.txt":   {Data: []byte("topic {{Language}}")},
	}
	agent := &Agent{llm: newLLM(cfg), store: newSessionStore(), prompts: fsys}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "web", "index.html"), []byte("<html>hi</html>"), 0o644); err != nil {
		t.Fatal(err)
	}

	return &App{cfg: cfg, tts: newTTS(cfg), agent: agent, webFS: os.DirFS(dir)}
}

func doRequest(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return m
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"k": "v"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if got := decodeJSON(t, rec)["k"]; got != "v" {
		t.Fatalf("k = %v", got)
	}
}

func TestHTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	httpError(rec, http.StatusBadRequest, "nope")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := decodeJSON(t, rec)["error"]; got != "nope" {
		t.Fatalf("error = %v", got)
	}
}

func TestReadJSON(t *testing.T) {
	var dst struct {
		Text string `json:"text"`
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"text":"hi"}`))
	if err := readJSON(rec, req, &dst); err != nil {
		t.Fatal(err)
	}
	if dst.Text != "hi" {
		t.Fatalf("Text = %q", dst.Text)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	if err := readJSON(rec, req, &dst); err == nil {
		t.Fatal("readJSON should error on invalid JSON")
	}
}

func TestWithCORS(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h := withCORS(inner)

	rec := doRequest(h, http.MethodGet, "/", "")
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("CORS origin = %q", got)
	}

	rec = doRequest(h, http.MethodOptions, "/", "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status = %d", rec.Code)
	}
}

func TestWithLogging(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	rec := doRequest(withLogging(inner), http.MethodGet, "/", "")
	if !called {
		t.Fatal("withLogging did not call next")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestProviderAvailable(t *testing.T) {
	a := &App{cfg: &Config{}}
	for _, p := range []string{"deepseek", "openai", "anthropic"} {
		if a.providerAvailable(p) {
			t.Fatalf("providerAvailable(%q) should be false without key", p)
		}
	}
	for _, p := range []string{"ollama", "mock", "unknown"} {
		if !a.providerAvailable(p) {
			t.Fatalf("providerAvailable(%q) should be true", p)
		}
	}

	a = &App{cfg: &Config{DeepSeekKey: "x", OpenAIKey: "y", AnthropicKey: "z"}}
	for _, p := range []string{"deepseek", "openai", "anthropic"} {
		if !a.providerAvailable(p) {
			t.Fatalf("providerAvailable(%q) should be true with key", p)
		}
	}
}

func TestHandleHealth(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodGet, "/api/health", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if decodeJSON(t, rec)["ok"] != true {
		t.Fatal("ok != true")
	}
}

func TestHandleLanguages(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodGet, "/api/languages", "")
	m := decodeJSON(t, rec)
	if langs, _ := m["languages"].([]any); len(langs) != 3 {
		t.Fatalf("languages = %d, want 3", len(langs))
	}
}

func TestHandleModels(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodGet, "/api/models", "")
	m := decodeJSON(t, rec)
	if models, _ := m["models"].([]any); len(models) != len(modelRegistry) {
		t.Fatalf("models = %d, want %d", len(models), len(modelRegistry))
	}
	if m["default"] != "mock-tutor" {
		t.Fatalf("default = %v", m["default"])
	}
}

func TestHandleConfig(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodGet, "/api/config", "")
	if got := decodeJSON(t, rec)["sttSilenceMs"]; got != float64(5000) {
		t.Fatalf("sttSilenceMs = %v", got)
	}
}

func TestSessionFlow(t *testing.T) {
	app := newTestApp(t)

	rec := doRequest(app.routes(), http.MethodPost, "/api/session/start", `{"language":"fr","model":"mock-tutor"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status = %d body=%s", rec.Code, rec.Body.String())
	}
	sid, _ := decodeJSON(t, rec)["sessionId"].(string)
	if sid == "" {
		t.Fatal("sessionId is empty")
	}

	rec = doRequest(app.routes(), http.MethodPost, "/api/session/message", `{"sessionId":"`+sid+`","text":"bonjour"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("message status = %d", rec.Code)
	}
	if decodeJSON(t, rec)["reply"] == "" {
		t.Fatal("reply is empty")
	}

	rec = doRequest(app.routes(), http.MethodPost, "/api/session/end", `{"sessionId":"`+sid+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("end status = %d", rec.Code)
	}
	if decodeJSON(t, rec)["goodbye"] == "" {
		t.Fatal("goodbye is empty")
	}
}

func TestHandleStartBadLanguage(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodPost, "/api/session/start", `{"language":"de","model":"mock-tutor"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleMessageEmptyText(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodPost, "/api/session/start", `{"language":"en","model":"mock-tutor"}`)
	sid := decodeJSON(t, rec)["sessionId"].(string)

	rec = doRequest(app.routes(), http.MethodPost, "/api/session/message", `{"sessionId":"`+sid+`","text":"  "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleTTSError(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodPost, "/api/tts", `{"text":"hi","lang":"en"}`)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRoutesServesIndex(t *testing.T) {
	app := newTestApp(t)
	rec := doRequest(app.routes(), http.MethodGet, "/", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html>hi</html>") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestValidatePrompts(t *testing.T) {
	full := fstest.MapFS{
		"tutor_system.txt": {Data: []byte("a")},
		"topic_system.txt": {Data: []byte("b")},
		"topic_user.txt":   {Data: []byte("c")},
	}
	if _, err := validatePrompts(full, "test"); err != nil {
		t.Fatalf("validatePrompts(full) = %v", err)
	}

	partial := fstest.MapFS{"tutor_system.txt": {Data: []byte("a")}}
	if _, err := validatePrompts(partial, "test"); err == nil {
		t.Fatal("validatePrompts(partial) should fail on missing files")
	}
}

func TestResolvePromptsEmbedded(t *testing.T) {
	embedded := fstest.MapFS{
		"prompts/tutor_system.txt": {Data: []byte("a")},
		"prompts/topic_system.txt": {Data: []byte("b")},
		"prompts/topic_user.txt":   {Data: []byte("c")},
	}
	fsys, err := resolvePrompts("", embedded)
	if err != nil {
		t.Fatalf("resolvePrompts embedded: %v", err)
	}
	data, err := fs.ReadFile(fsys, "tutor_system.txt")
	if err != nil || string(data) != "a" {
		t.Fatalf("read = %q, %v", data, err)
	}
}

func TestResolvePromptsDir(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"tutor_system.txt": "a",
		"topic_system.txt": "b",
		"topic_user.txt":   "c",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fsys, err := resolvePrompts(dir, nil)
	if err != nil {
		t.Fatalf("resolvePrompts dir: %v", err)
	}
	data, err := fs.ReadFile(fsys, "topic_system.txt")
	if err != nil || string(data) != "b" {
		t.Fatalf("read = %q, %v", data, err)
	}
}

func TestResolvePromptsMissingDir(t *testing.T) {
	if _, err := resolvePrompts(filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Fatal("resolvePrompts should fail for a missing directory")
	}
}
