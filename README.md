# Polyglot — voice language tutor

A small, self-contained web app for practicing spoken **English / Spanish / French**
by having a voice conversation with an AI tutor. You talk into your microphone,
the tutor replies — in the target language, out loud.

The server is written in **Go** (standard library only — no external dependencies).

## Features

- **Web app** with a language selector (English · Español · Français) and a model selector.
- **Start discussion** / **End discussion** buttons.
- **Microphone input** — speech recognition via the browser's Web Speech API.
- **Voice from the server** — text-to-speech is generated server-side and streamed
  back as audio (keyless Google Translate TTS by default, or OpenAI TTS).
- **Go agent backend** that picks a **new topic every session** and keeps the
  conversation going with a friendly, short-reply tutor persona.
- **Multiple models**, defaulting to `deepseek-v4-pro`:
  - DeepSeek (`deepseek-v4-pro`, `deepseek-chat`)
  - OpenAI (`gpt-4o`, `gpt-4o-mini`)
  - Anthropic (`claude-3-5-haiku-latest`, `claude-3-7-sonnet-latest`)
  - Ollama (`llama3.2`, local)
  - A built-in offline **demo tutor** (no API key required)

## Run it

Requires **Go 1.22+**.

```bash
go build -o ailanglearn .
./ailanglearn
```

Then open <http://localhost:8080>. There is nothing else to install — the web
assets are embedded in the binary.

> The demo tutor works with **no API key**. Set the keys below to use real models.

## Configuration

All settings come from environment variables (see `.env.example`):

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `LLM_MODEL` | `deepseek-v4-pro` | Default model shown in the UI |
| `DEEPSEEK_API_BASE` | `https://api.deepseek.com` | DeepSeek base URL |
| `DEEPSEEK_API_KEY` | — | DeepSeek API key |
| `OPENAI_API_BASE` | `https://api.openai.com/v1` | OpenAI base URL |
| `OPENAI_API_KEY` | — | OpenAI API key |
| `ANTHROPIC_API_BASE` | `https://api.anthropic.com` | Anthropic base URL |
| `ANTHROPIC_API_KEY` | — | Anthropic API key |
| `OLLAMA_BASE` | `http://localhost:11434` | Local Ollama base URL |
| `TTS_PROVIDER` | `google` | `google` (keyless) · `openai` · `mock` |
| `TTS_API_BASE` / `TTS_API_KEY` / `TTS_MODEL` / `TTS_VOICE` | — | Used when `TTS_PROVIDER=openai` |

### A note on `deepseek-v4-pro`

`deepseek-v4-pro` is the default model id. The server treats DeepSeek as an
OpenAI-compatible endpoint, so you can point it at any compatible gateway:

```bash
# DeepSeek's public API (use their model name):
DEEPSEEK_API_KEY=sk-... LLM_MODEL=deepseek-chat ./ailanglearn

# Any OpenAI-compatible gateway serving deepseek-v4-pro:
DEEPSEEK_API_BASE=https://your-gateway DEEPSEEK_API_KEY=... LLM_MODEL=deepseek-v4-pro ./ailanglearn
```

If a selected model's provider has no key configured, the server automatically
falls back to the offline demo tutor and the UI shows a small **demo** badge.

## How it works

- `POST /api/session/start` creates a session and generates a **fresh topic**
  (the agent avoids recently used topics), returning a greeting in the target
  language that is spoken aloud.
- `POST /api/session/message` sends your transcribed speech to the model and
  returns the tutor's short reply.
- `POST /api/session/end` closes the session.
- `POST /api/tts` renders any text to audio on the server.

### Browser notes

- **Speech recognition** uses the browser's Web Speech API, which is available in
  Chrome and Edge (and needs an internet connection). The app shows a typed-input
  fallback everywhere, so it works even without a microphone.
- If server TTS is unavailable, the app falls back to the browser's built-in
  speech synthesis.

## Project layout

```
main.go                  entry point (embeds web/ and prompts/)
internal/voicechat/
  run.go                 bootstrap: config, wiring, server startup
  config.go              env config, language/model registry
  server.go              HTTP API + static file serving
  agent.go               sessions, tutor persona, topic generation
  llm.go                 model providers (DeepSeek/OpenAI/Anthropic/Ollama/demo)
  tts.go                 server-side text-to-speech
  topics.go              curated fallback topics per language
  prompts.go             prompt template loading and rendering
web/                     frontend (index.html, style.css, app.js)
prompts/                 prompt templates (*.txt)
```
