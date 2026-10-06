package credentials

import (
	"bytes"
	"context"
	"testing"
)

func TestSealRoundTripDoesNotStoreTheSecret(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	provider, err := NewEnvKey(key)
	if err != nil {
		t.Fatal(err)
	}
	sealer := NewSealer(provider)
	const secret = "sk-live-do-not-store"

	sealed, err := sealer.Seal(context.Background(), []byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed.Ciphertext, []byte(secret)) || bytes.Contains(sealed.WrappedKey, []byte(secret)) {
		t.Fatal("secret appeared in the sealed blob")
	}
	if sealed.Fingerprint == "" || sealed.KeyVersion != 1 {
		t.Fatalf("sealed = %+v", sealed)
	}

	opened, err := sealer.Open(context.Background(), sealed)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened) != secret {
		t.Fatalf("opened = %q", opened)
	}
}

func TestTwoSealsUseDifferentDataKeys(t *testing.T) {
	provider, err := NewEnvKey(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealer := NewSealer(provider)
	first, err := sealer.Seal(context.Background(), []byte("same-secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := sealer.Seal(context.Background(), []byte("same-secret-value"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.WrappedKey, second.WrappedKey) {
		t.Fatal("expected a new data key for each seal")
	}
}
