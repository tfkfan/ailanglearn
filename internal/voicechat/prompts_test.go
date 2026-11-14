package voicechat

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestRequiredPromptFiles(t *testing.T) {
	want := []string{"tutor_system.txt", "topic_system.txt", "topic_user.txt"}
	if len(requiredPromptFiles) != len(want) {
		t.Fatalf("requiredPromptFiles = %v, want %v", requiredPromptFiles, want)
	}
	for _, w := range want {
		found := false
		for _, have := range requiredPromptFiles {
			if have == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("requiredPromptFiles missing %q", w)
		}
	}
}

func TestLoadPrompt(t *testing.T) {
	fsys := fstest.MapFS{"hello.txt": {Data: []byte("  hi there  \n")}}
	if got := loadPrompt(fsys, "hello.txt"); got != "hi there" {
		t.Fatalf("loadPrompt = %q, want %q", got, "hi there")
	}
	if got := loadPrompt(fsys, "missing.txt"); got != "" {
		t.Fatalf("loadPrompt(missing) = %q, want empty", got)
	}
}

func TestRender(t *testing.T) {
	out := render("Hello {{Name}} and {{Name}}", map[string]string{"Name": "Ana"})
	if out != "Hello Ana and Ana" {
		t.Fatalf("render = %q", out)
	}
	if strings.Contains(out, "{{") {
		t.Fatal("render left a placeholder behind")
	}

	out = render("Hi {{Unknown}}", map[string]string{})
	if out != "Hi {{Unknown}}" {
		t.Fatalf("unknown placeholder should be untouched, got %q", out)
	}
}
