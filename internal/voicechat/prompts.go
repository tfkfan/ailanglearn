package voicechat

import (
	"io/fs"
	"log"
	"strings"
)

// loadPrompt reads a prompt template from the embedded prompts directory.
func loadPrompt(fsys fs.FS, name string) string {
	data, err := fs.ReadFile(fsys, "prompts/"+name)
	if err != nil {
		log.Printf("missing prompt template %s: %v", name, err)
		return ""
	}
	return strings.TrimSpace(string(data))
}

// render substitutes {{Name}} placeholders in a prompt template.
func render(tpl string, vars map[string]string) string {
	for k, v := range vars {
		tpl = strings.ReplaceAll(tpl, "{{"+k+"}}", v)
	}
	return tpl
}
