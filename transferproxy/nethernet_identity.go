package proxy

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var netherNetIdentityMu sync.Mutex

func ensureNetherNetIdentity(path string) (*ecdsa.PrivateKey, bool, error) {
	if path == "" {
		path = filepath.Join("Keys", "identity.pem")
	}

	netherNetIdentityMu.Lock()
	defer netherNetIdentityMu.Unlock()

	exists, err := fileExists(path)
	if err != nil {
		return nil, false, fmt.Errorf("proxy: inspect NetherNet identity: %w", err)
	}
	if exists {
		key, err := readNetherNetIdentity(path)
		if err != nil {
			return nil, false, err
		}
		return key, false, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, false, fmt.Errorf("proxy: create NetherNet identity directory: %w", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, false, fmt.Errorf("proxy: generate NetherNet identity key: %w", err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, false, fmt.Errorf("proxy: encode NetherNet identity key: %w", err)
	}
	if err := writeNewFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: encoded,
	}), 0o600); err != nil {
		if errors.Is(err, os.ErrExist) {
			key, readErr := readNetherNetIdentity(path)
			return key, false, readErr
		}
		return nil, false, fmt.Errorf("proxy: write NetherNet identity: %w", err)
	}
	return key, true, nil
}

func readNetherNetIdentity(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("proxy: read NetherNet identity: %w", err)
	}
	block, rest := pem.Decode(data)
	if block == nil || len(bytes.TrimSpace(rest)) != 0 {
		return nil, fmt.Errorf("proxy: parse NetherNet identity PEM")
	}

	var parsed any
	switch block.Type {
	case "PRIVATE KEY":
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		parsed, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("proxy: unsupported NetherNet identity PEM block %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("proxy: parse NetherNet identity key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("proxy: NetherNet identity key is %T, want ECDSA", parsed)
	}
	if key.Curve.Params().Name != elliptic.P384().Params().Name {
		return nil, fmt.Errorf("proxy: NetherNet identity key uses %s, want P-384", key.Curve.Params().Name)
	}
	return key, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func writeNewFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}
