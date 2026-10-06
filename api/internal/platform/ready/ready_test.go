package ready

import "testing"

func TestLatestMatchesEmbeddedMigrations(t *testing.T) {
	latest, err := Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest != 4 {
		t.Fatalf("latest = %d, want 4", latest)
	}
}
