package voicechat

import "testing"

func TestTopicSeeds(t *testing.T) {
	for _, lang := range []string{"en", "es", "fr"} {
		seeds := topicSeeds[lang]
		if len(seeds) == 0 {
			t.Fatalf("no topic seeds for %q", lang)
		}
		seen := map[string]bool{}
		for _, s := range seeds {
			if s.topic == "" || s.greeting == "" {
				t.Fatalf("seed %+v has an empty field (lang %q)", s, lang)
			}
			if seen[s.topic] {
				t.Fatalf("duplicate topic %q in lang %q", s.topic, lang)
			}
			seen[s.topic] = true
		}
	}
}
