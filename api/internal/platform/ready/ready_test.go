package ready

import "testing"

func TestLatestMatchesEmbeddedMigrations(t *testing.T) {
	latest, err := Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest != 2 {
		t.Fatalf("latest = %d, want 2", latest)
	}
}
