package connectors

import "testing"

func TestRegistryListsKinds(t *testing.T) {
	registry := NewRegistry(
		Disabled{KindName: "openai", DisplayName: "OpenAI"},
		Disabled{KindName: "anthropic", DisplayName: "Anthropic"},
	)
	kinds := registry.Kinds()
	if len(kinds) != 2 || kinds[0] != "anthropic" || kinds[1] != "openai" {
		t.Fatalf("kinds = %v", kinds)
	}
	if _, ok := registry.Get("missing"); ok {
		t.Fatal("expected a missing kind to be absent")
	}
}
