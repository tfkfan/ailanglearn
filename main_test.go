package main

import "testing"

func TestEmbeddedAssets(t *testing.T) {
	if _, err := webFS.ReadFile("web/index.html"); err != nil {
		t.Fatalf("web/index.html not embedded: %v", err)
	}
	for _, name := range []string{"tutor_system.txt", "topic_system.txt", "topic_user.txt"} {
		if _, err := promptsFS.ReadFile("prompts/" + name); err != nil {
			t.Fatalf("prompts/%s not embedded: %v", name, err)
		}
	}
}
