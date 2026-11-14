package voicechat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"sync"
	"time"
)

type Session struct {
	ID       string    `json:"id"`
	Language string    `json:"language"`
	Model    string    `json:"model"`
	Topic    string    `json:"topic"`
	Opening  string    `json:"opening"`
	Created  time.Time `json:"created"`
	Active   bool      `json:"active"`

	mu    sync.Mutex
	Turns []Message
}

type sessionStore struct {
	mu     sync.Mutex
	m      map[string]*Session
	recent map[string][]string
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		m:      map[string]*Session{},
		recent: map[string][]string{},
	}
}

func (s *sessionStore) put(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[sess.ID] = sess
}

func (s *sessionStore) get(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[id]
	if !ok {
		return nil, errors.New("session not found")
	}
	return sess, nil
}

func (s *sessionStore) recentTopics(lang string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.recent[lang]...)
}

func (s *sessionStore) rememberTopic(lang, topic string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list := s.recent[lang]
	for _, t := range list {
		if strings.EqualFold(t, topic) {
			return
		}
	}
	list = append(list, topic)
	if len(list) > 6 {
		list = list[len(list)-6:]
	}
	s.recent[lang] = list
}

type Agent struct {
	llm     *LLM
	store   *sessionStore
	prompts fs.FS
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *Agent) Start(lang, model string) (*Session, error) {
	l, ok := langByCode(lang)
	if !ok {
		return nil, fmt.Errorf("unsupported language %q", lang)
	}
	if model == "" {
		model = "deepseek-v4-pro"
	}

	sess := &Session{
		ID:       newID(),
		Language: lang,
		Model:    model,
		Created:  time.Now(),
		Active:   true,
	}

	topic, opening := a.pickTopic(l, model)
	sess.Topic = topic
	sess.Opening = opening
	a.store.put(sess)
	a.store.rememberTopic(lang, topic)
	return sess, nil
}

func (a *Agent) pickTopic(l Lang, model string) (topic, opening string) {
	if topic, opening, ok := a.llmTopic(l, model); ok {
		return topic, opening
	}
	return a.seedTopic(l)
}

func (a *Agent) llmTopic(l Lang, model string) (string, string, bool) {
	recent := a.store.recentTopics(l.Code)
	sys := loadPrompt(a.prompts, "topic_system.txt")
	prompt := render(loadPrompt(a.prompts, "topic_user.txt"), map[string]string{
		"Language":     l.Name,
		"RecentTopics": joinOrNone(recent),
	})

	raw, demo, err := a.llm.Chat(model, l.Code, sys, []Message{{Role: "user", Content: prompt}})
	if err != nil || demo {
		return "", "", false
	}

	var parsed struct {
		Topic    string `json:"topic"`
		Greeting string `json:"greeting"`
	}
	if err := json.Unmarshal([]byte(stripFences(raw)), &parsed); err != nil {
		return "", "", false
	}
	topic := strings.TrimSpace(parsed.Topic)
	greeting := strings.TrimSpace(parsed.Greeting)
	if topic == "" || greeting == "" {
		return "", "", false
	}
	return topic, greeting, true
}

func (a *Agent) seedTopic(l Lang) (string, string) {
	seeds := topicSeeds[l.Code]
	if len(seeds) == 0 {
		return "General conversation", "Hello! Let's have a conversation. What would you like to talk about today?"
	}
	recent := a.store.recentTopics(l.Code)
	for _, s := range seeds {
		if !containsFold(recent, s.topic) {
			return s.topic, s.greeting
		}
	}
	s := seeds[len(recent)%len(seeds)]
	return s.topic, s.greeting
}

func (a *Agent) Send(sessionID, text string) (reply string, demo bool, err error) {
	sess, err := a.store.get(sessionID)
	if err != nil {
		return "", false, err
	}

	userMsg := Message{Role: "user", Content: strings.TrimSpace(text)}

	sess.mu.Lock()
	if !sess.Active {
		sess.mu.Unlock()
		return "", false, errors.New("session has ended")
	}
	sess.Turns = append(sess.Turns, userMsg)
	history := append([]Message{}, sess.Turns...)
	sess.mu.Unlock()

	l, _ := langByCode(sess.Language)
	sys := a.tutorPrompt(l, sess.Topic, sess.Opening)
	reply, demo, err = a.llm.Chat(sess.Model, l.Code, sys, history)
	if err != nil {
		return "", demo, err
	}
	reply = strings.TrimSpace(reply)
	if reply == "" {
		reply = mockReply(l.Code, history)
		demo = true
	}

	sess.mu.Lock()
	sess.Turns = append(sess.Turns, Message{Role: "assistant", Content: reply})
	sess.mu.Unlock()
	return reply, demo, nil
}

func (a *Agent) End(sessionID string) (string, error) {
	sess, err := a.store.get(sessionID)
	if err != nil {
		return "", err
	}
	sess.mu.Lock()
	sess.Active = false
	sess.mu.Unlock()
	return goodbye(sess.Language), nil
}

func (a *Agent) tutorPrompt(l Lang, topic, opening string) string {
	return render(loadPrompt(a.prompts, "tutor_system.txt"), map[string]string{
		"Language": l.Name,
		"Topic":    topic,
		"Opening":  opening,
	})
}

func goodbye(lang string) string {
	switch lang {
	case "es":
		return "¡Muy bien! Eso es todo por hoy. ¡Hasta la próxima!"
	case "fr":
		return "Très bien ! C'est tout pour aujourd'hui. À la prochaine !"
	default:
		return "Great work today! That's all for now — see you next time."
	}
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none yet)"
	}
	return strings.Join(items, ", ")
}

func stripFences(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func containsFold(list []string, s string) bool {
	for _, x := range list {
		if strings.EqualFold(x, s) {
			return true
		}
	}
	return false
}
