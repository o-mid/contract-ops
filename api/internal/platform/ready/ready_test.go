package ready

import "testing"

func TestLatestMatchesEmbeddedMigrations(t *testing.T) {
	latest, err := Latest()
	if err != nil {
		t.Fatal(err)
	}
	if latest != 3 {
		t.Fatalf("latest = %d, want 3", latest)
	}
}
