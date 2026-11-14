package voicechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TTS converts text to speech audio on the server.
type TTS struct {
	provider string
	http     *http.Client
	base     string
	key      string
	model    string
	voice    string
}

func newTTS(cfg *Config) *TTS {
	return &TTS{
		provider: cfg.TTSProvider,
		http:     &http.Client{Timeout: 30 * time.Second},
		base:     cfg.TTSBase,
		key:      cfg.TTSKey,
		model:    cfg.TTSModel,
		voice:    cfg.TTSVoice,
	}
}

// Speak returns audio bytes and a MIME type.
func (t *TTS) Speak(text, lang string) ([]byte, string, error) {
	switch t.provider {
	case "openai":
		if t.key == "" {
			return nil, "", fmt.Errorf("TTS_API_KEY is not set")
		}
		return t.openAI(text)
	case "mock":
		return nil, "", fmt.Errorf("server tts disabled (mock)")
	default: // google
		data, err := t.google(text, lang)
		if err != nil {
			return nil, "", err
		}
		return data, "audio/mpeg", nil
	}
}

func (t *TTS) openAI(text string) ([]byte, string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": t.model,
		"voice": t.voice,
		"input": text,
	})
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(t.base, "/")+"/audio/speech", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.key)
	resp, err := t.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("tts upstream %d: %s", resp.StatusCode, truncate(string(data), 200))
	}
	return data, "audio/mpeg", nil
}

// google uses the keyless Google Translate TTS endpoint. Longer text is split
// on sentence boundaries and the MP3 frames are concatenated, which browsers
// and audio players decode fine.
func (t *TTS) google(text, lang string) ([]byte, error) {
	var buf bytes.Buffer
	for _, chunk := range splitTTS(text, 180) {
		q := url.Values{}
		q.Set("ie", "UTF-8")
		q.Set("q", chunk)
		q.Set("tl", lang)
		q.Set("client", "tw-ob")
		q.Set("total", "1")
		q.Set("idx", "0")
		q.Set("textlen", fmt.Sprint(len(chunk)))
		u := "https://translate.google.com/translate_tts?" + q.Encode()

		req, err := http.NewRequest(http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0 Safari/537.36")
		resp, err := t.http.Do(req)
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("google tts status %d", resp.StatusCode)
		}
		buf.Write(data)
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("google tts returned no audio")
	}
	return buf.Bytes(), nil
}

func splitTTS(text string, max int) []string {
	text = strings.Join(strings.Fields(text), " ")
	var parts []string
	for _, sent := range splitSentences(text) {
		if len(sent) <= max {
			parts = append(parts, sent)
			continue
		}
		// Hard-split an over-long sentence on word boundaries.
		var cur strings.Builder
		for _, w := range strings.Fields(sent) {
			if cur.Len() > 0 && cur.Len()+len(w)+1 > max {
				parts = append(parts, cur.String())
				cur.Reset()
			}
			if cur.Len() > 0 {
				cur.WriteByte(' ')
			}
			cur.WriteString(w)
		}
		if cur.Len() > 0 {
			parts = append(parts, cur.String())
		}
	}
	if len(parts) == 0 {
		parts = append(parts, text)
	}
	return parts
}

func splitSentences(text string) []string {
	var parts []string
	start := 0
	for i, r := range text {
		if r == '.' || r == '!' || r == '?' || r == ';' || r == '\n' {
			parts = append(parts, strings.TrimSpace(text[start:i+1]))
			start = i + 1
		}
	}
	if rest := strings.TrimSpace(text[start:]); rest != "" {
		parts = append(parts, rest)
	}
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
