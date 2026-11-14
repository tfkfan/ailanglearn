package voicechat

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

// Run loads configuration, wires up the agent, and serves the web app until
// the process exits. webAssets and promptAssets are the embedded static files
// and prompt templates, normally provided by the main package via //go:embed.
func Run(webAssets, promptAssets fs.FS) {
	// Load variables from .env (in the working directory) before reading any
	// config. Existing environment variables take precedence and are not
	// overridden by godotenv.
	if err := godotenv.Load(); err != nil {
		log.Printf("note: could not load .env: %v", err)
	}

	cfg := loadConfig()
	llm := newLLM(cfg)
	tts := newTTS(cfg)
	agent := &Agent{llm: llm, store: newSessionStore(), prompts: promptAssets}
	app := &App{cfg: cfg, tts: tts, agent: agent, webFS: webAssets}

	logConfig(cfg)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: app.routes(),
	}

	log.Printf("Polyglot tutor listening on http://localhost:%s", cfg.Port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
