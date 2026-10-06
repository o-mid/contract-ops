// Package credentials seals secrets with a per-secret data key.
// The master key only wraps that data key, so it never encrypts the secret directly.
package credentials

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Sealed struct {
	Ciphertext  []byte
	WrappedKey  []byte
	KeyVersion  int
	Fingerprint string
}

// KeyProvider returns master keys. Current is used to seal. Key opens older versions.
type KeyProvider interface {
	Current(ctx context.Context) (version int, key []byte, err error)
	Key(ctx context.Context, version int) ([]byte, error)
}

type EnvKey struct {
	raw []byte
}

func NewEnvKey(raw []byte) (*EnvKey, error) {
	if len(raw) != 32 {
		return nil, fmt.Errorf("master key must be 32 bytes")
	}
	copied := make([]byte, 32)
	copy(copied, raw)
	return &EnvKey{raw: copied}, nil
}

func (k *EnvKey) Current(context.Context) (int, []byte, error) {
	return 1, k.raw, nil
}

func (k *EnvKey) Key(_ context.Context, version int) ([]byte, error) {
	if version != 1 {
		return nil, fmt.Errorf("unknown master key version %d", version)
	}
	return k.raw, nil
}

type Sealer struct {
	keys KeyProvider
}

func NewSealer(keys KeyProvider) *Sealer {
	return &Sealer{keys: keys}
}

func (s *Sealer) Seal(ctx context.Context, plaintext []byte) (Sealed, error) {
	version, master, err := s.keys.Current(ctx)
	if err != nil {
		return Sealed{}, err
	}

	dataKey := make([]byte, 32)
	if _, err := rand.Read(dataKey); err != nil {
		return Sealed{}, err
	}

	ciphertext, err := encrypt(dataKey, plaintext)
	if err != nil {
		return Sealed{}, err
	}
	wrapped, err := encrypt(master, dataKey)
	if err != nil {
		return Sealed{}, err
	}

	sum := sha256.Sum256(plaintext)
	return Sealed{
		Ciphertext:  ciphertext,
		WrappedKey:  wrapped,
		KeyVersion:  version,
		Fingerprint: hex.EncodeToString(sum[:]),
	}, nil
}

func (s *Sealer) Open(ctx context.Context, sealed Sealed) ([]byte, error) {
	master, err := s.keys.Key(ctx, sealed.KeyVersion)
	if err != nil {
		return nil, err
	}
	dataKey, err := decrypt(master, sealed.WrappedKey)
	if err != nil {
		return nil, fmt.Errorf("unwrap data key: %w", err)
	}
	plaintext, err := decrypt(dataKey, sealed.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("open secret: %w", err)
	}
	return plaintext, nil
}

func encrypt(key, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(key, sealed []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(sealed) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext is too short")
	}
	nonce := sealed[:gcm.NonceSize()]
	body := sealed[gcm.NonceSize():]
	return gcm.Open(nil, nonce, body, nil)
}
