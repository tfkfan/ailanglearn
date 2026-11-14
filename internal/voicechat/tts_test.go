package voicechat

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSplitSentences(t *testing.T) {
	got := splitSentences("Hello there. How are you? I'm fine!")
	want := []string{"Hello there.", "How are you?", "I'm fine!"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitSentences = %#v, want %#v", got, want)
	}
}

func TestSplitTTS(t *testing.T) {
	got := splitTTS("Short. Another one.", 100)
	want := []string{"Short.", "Another one."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitTTS = %#v, want %#v", got, want)
	}

	long := "aaaaaaaaaa bbbbbbbbbb cccccccccc"
	chunks := splitTTS(long, 15)
	for _, c := range chunks {
		if len(c) > 15 {
			t.Fatalf("chunk %q exceeds max length", c)
		}
	}
	if len(chunks) != 3 {
		t.Fatalf("chunks = %#v, want 3 pieces", chunks)
	}
}

func TestSpeakMockProvider(t *testing.T) {
	tts := newTTS(&Config{TTSProvider: "mock"})
	if _, _, err := tts.Speak("hi", "en"); err == nil {
		t.Fatal("expected error for mock tts provider")
	}
}

func TestSpeakOpenAINoKey(t *testing.T) {
	tts := newTTS(&Config{TTSProvider: "openai"})
	if _, _, err := tts.Speak("hi", "en"); err == nil {
		t.Fatal("expected error when TTS_API_KEY is unset")
	}
}

func TestTTSOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/audio/speech" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer k" {
			t.Errorf("auth = %q", got)
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("MP3DATA"))
	}))
	defer srv.Close()

	tts := &TTS{http: srv.Client(), base: srv.URL, key: "k", model: "tts-1", voice: "alloy"}
	data, mime, err := tts.openAI("hello")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "MP3DATA" {
		t.Fatalf("data = %q", data)
	}
	if mime != "audio/mpeg" {
		t.Fatalf("mime = %q", mime)
	}
}
