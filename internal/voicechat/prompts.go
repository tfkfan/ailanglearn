package voicechat

import (
	"io/fs"
	"log"
	"strings"
)

var requiredPromptFiles = []string{
	"tutor_system.txt",
	"topic_system.txt",
	"topic_user.txt",
}

func loadPrompt(fsys fs.FS, name string) string {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		log.Printf("missing prompt template %s: %v", name, err)
		return ""
	}
	return strings.TrimSpace(string(data))
}

func render(tpl string, vars map[string]string) string {
	for k, v := range vars {
		tpl = strings.ReplaceAll(tpl, "{{"+k+"}}", v)
	}
	return tpl
}
