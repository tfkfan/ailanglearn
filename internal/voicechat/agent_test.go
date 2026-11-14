package voicechat

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func newTestAgent() *Agent {
	fsys := fstest.MapFS{
		"tutor_system.txt": {Data: []byte("You are a {{Language}} tutor. Topic: {{Topic}}. Opening: {{Opening}}.")},
		"topic_system.txt": {Data: []byte("Reply with JSON only.")},
		"topic_user.txt":   {Data: []byte("Language: {{Language}}. Avoid: {{RecentTopics}}.")},
	}
	return &Agent{llm: newLLM(&Config{}), store: newSessionStore(), prompts: fsys}
}

func TestNewID(t *testing.T) {
	a := newID()
	b := newID()
	if a == "" || b == "" {
		t.Fatal("newID returned empty")
	}
	if a == b {
		t.Fatal("newID returned duplicates")
	}
	if len(a) != 32 {
		t.Fatalf("newID length = %d, want 32", len(a))
	}
}

func TestJoinOrNone(t *testing.T) {
	if got := joinOrNone(nil); got != "(none yet)" {
		t.Fatalf("joinOrNone(nil) = %q", got)
	}
	if got := joinOrNone([]string{"a", "b"}); got != "a, b" {
		t.Fatalf("joinOrNone = %q", got)
	}
}

func TestStripFences(t *testing.T) {
	if got := stripFences("```json\n{\"a\":1}\n```"); got != "{\"a\":1}" {
		t.Fatalf("stripFences = %q", got)
	}
	if got := stripFences(`{"a":1}`); got != `{"a":1}` {
		t.Fatalf("stripFences(plain) = %q", got)
	}
}

func TestContainsFold(t *testing.T) {
	if !containsFold([]string{"Travel", "Food"}, "travel") {
		t.Fatal("containsFold should match case-insensitively")
	}
	if containsFold([]string{"Travel"}, "Music") {
		t.Fatal("containsFold should not match")
	}
}

func TestGoodbye(t *testing.T) {
	for _, lang := range []string{"en", "es", "fr"} {
		if got := goodbye(lang); got == "" {
			t.Fatalf("goodbye(%q) is empty", lang)
		}
	}
	if got := goodbye("xx"); got == "" {
		t.Fatal("goodbye(xx) is empty")
	}
}

func TestTutorPrompt(t *testing.T) {
	agent := newTestAgent()
	out := agent.tutorPrompt(Lang{Name: "Spanish"}, "Travel", "Hola")
	for _, want := range []string{"Spanish", "Travel", "Hola"} {
		if !strings.Contains(out, want) {
			t.Fatalf("tutorPrompt missing %q: %q", want, out)
		}
	}
	if strings.Contains(out, "{{") {
		t.Fatalf("tutorPrompt left a placeholder: %q", out)
	}
}

func TestSeedTopic(t *testing.T) {
	agent := newTestAgent()
	for _, code := range []string{"en", "es", "fr"} {
		l, _ := langByCode(code)
		topic, opening := agent.seedTopic(l)
		if topic == "" || opening == "" {
			t.Fatalf("seedTopic(%q) returned empty", code)
		}
		found := false
		for _, s := range topicSeeds[code] {
			if s.topic == topic {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("seedTopic(%q) returned unknown topic %q", code, topic)
		}
	}
}

func TestSessionStore(t *testing.T) {
	s := newSessionStore()
	if _, err := s.get("missing"); err == nil {
		t.Fatal("get(missing) should error")
	}

	sess := &Session{ID: "abc", Active: true}
	s.put(sess)
	got, err := s.get("abc")
	if err != nil || got != sess {
		t.Fatalf("get = %+v, %v", got, err)
	}

	s.rememberTopic("en", "Travel")
	s.rememberTopic("en", "travel")
	if got := s.recentTopics("en"); len(got) != 1 {
		t.Fatalf("recentTopics after dedup = %v", got)
	}
	for i := 0; i < 10; i++ {
		s.rememberTopic("en", fmt.Sprintf("topic-%d", i))
	}
	if got := s.recentTopics("en"); len(got) > 6 {
		t.Fatalf("recentTopics grew beyond cap: %d", len(got))
	}
}

func TestStart(t *testing.T) {
	agent := newTestAgent()
	sess, err := agent.Start("es", "mock-tutor")
	if err != nil {
		t.Fatal(err)
	}
	if sess.ID == "" || !sess.Active || sess.Topic == "" || sess.Opening == "" {
		t.Fatalf("session = %+v", sess)
	}
	if got, err := agent.store.get(sess.ID); err != nil || got != sess {
		t.Fatalf("stored session mismatch: %v", err)
	}
	recent := agent.store.recentTopics("es")
	if len(recent) != 1 || recent[0] != sess.Topic {
		t.Fatalf("recentTopics = %v", recent)
	}
}

func TestStartBadLanguage(t *testing.T) {
	agent := newTestAgent()
	if _, err := agent.Start("de", "mock-tutor"); err == nil {
		t.Fatal("Start(de) should error")
	}
}

func TestSend(t *testing.T) {
	agent := newTestAgent()
	sess, _ := agent.Start("en", "mock-tutor")

	reply, demo, err := agent.Send(sess.ID, "I like music")
	if err != nil {
		t.Fatal(err)
	}
	if !demo || reply == "" {
		t.Fatalf("reply = %q, demo = %v", reply, demo)
	}

	got, _ := agent.store.get(sess.ID)
	if len(got.Turns) != 2 {
		t.Fatalf("turns = %d, want 2", len(got.Turns))
	}
	if got.Turns[0].Role != "user" || got.Turns[1].Role != "assistant" {
		t.Fatalf("turns = %+v", got.Turns)
	}
}

func TestSendMissingSession(t *testing.T) {
	agent := newTestAgent()
	if _, _, err := agent.Send("nope", "hi"); err == nil {
		t.Fatal("Send(missing) should error")
	}
}

func TestEndAndSendAfterEnd(t *testing.T) {
	agent := newTestAgent()
	sess, _ := agent.Start("en", "mock-tutor")

	msg, err := agent.End(sess.ID)
	if err != nil {
		t.Fatal(err)
	}
	if msg == "" {
		t.Fatal("goodbye is empty")
	}
	if _, _, err := agent.Send(sess.ID, "hi"); err == nil {
		t.Fatal("Send after End should error")
	}
}
