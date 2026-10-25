// Package secretbox encrypts secrets (AES-256-GCM) with a key from a file. The file is created
// with random content on first use (0600); back it up with the database, or secrets are lost.
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
)

type Box struct {
	path string
	mu   sync.Mutex
	aead cipher.AEAD
}

func New(keyFile string) *Box {
	return &Box{path: keyFile}
}

func (b *Box) load(create bool) (cipher.AEAD, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.aead != nil {
		return b.aead, nil
	}
	key, err := os.ReadFile(b.path)
	if errors.Is(err, os.ErrNotExist) && create {
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		// O_EXCL: a key created in parallel is never overwritten
		f, err := os.OpenFile(b.path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create key file: %w", err)
		}
		_, werr := f.Write(key)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			os.Remove(b.path)
			return nil, fmt.Errorf("write key file: %w", werr)
		}
	} else if err != nil {
		return nil, fmt.Errorf("read key file: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("key file %s must hold 32 bytes", b.path)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if b.aead, err = cipher.NewGCM(block); err != nil {
		return nil, err
	}
	return b.aead, nil
}

func (b *Box) Encrypt(plain string) (string, error) {
	aead, err := b.load(true)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (b *Box) Decrypt(sealed string) (string, error) {
	aead, err := b.load(false)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil || len(raw) < aead.NonceSize() {
		return "", errors.New("malformed secret")
	}
	plain, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("cannot decrypt secret (wrong or missing key file)")
	}
	return string(plain), nil
}
