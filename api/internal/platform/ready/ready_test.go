package ready

import "testing"

func TestLatestMatchesEmbeddedMigrations(t *testing.T) {
	latest, err := Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest != 6 {
		t.Fatalf("latest = %d, want 6", latest)
	}
}
