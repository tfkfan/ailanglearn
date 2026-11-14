package main

import (
	"embed"

	"ailanglearn/internal/voicechat"
)

//go:embed web
var webFS embed.FS

//go:embed prompts
var promptsFS embed.FS

func main() {
	voicechat.Run(webFS, promptsFS)
}
