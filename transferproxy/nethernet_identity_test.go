package proxy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/df-mc/go-nethernet"
)

func TestEnsureNetherNetIdentityGeneratesAndReusesKey(t *testing.T) {
	identityFile := filepath.Join(t.TempDir(), "keys", "identity.pem")
	key, generated, err := ensureNetherNetIdentity(identityFile)
	if err != nil {
		t.Fatal(err)
	}
	if !generated {
		t.Fatal("new identity was not reported as generated")
	}
	if key.Curve.Params().Name != elliptic.P384().Params().Name {
		t.Fatalf("generated key curve = %s, want P-384", key.Curve.Params().Name)
	}
	if _, err := nethernet.ServerIdentity(key, "self"); err != nil {
		t.Fatalf("generated key cannot issue a NetherNet server identity: %v", err)
	}
	keyBefore, err := os.ReadFile(identityFile)
	if err != nil {
		t.Fatal(err)
	}
	reusedKey, generated, err := ensureNetherNetIdentity(identityFile)
	if err != nil {
		t.Fatal(err)
	}
	if generated {
		t.Fatal("existing identity was reported as newly generated")
	}
	keyAfter, err := os.ReadFile(identityFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(keyBefore) != string(keyAfter) {
		t.Fatal("existing NetherNet identity was replaced")
	}
	if key.X.Cmp(reusedKey.X) != 0 || key.Y.Cmp(reusedKey.Y) != 0 {
		t.Fatal("reloaded NetherNet identity does not match the generated key")
	}
}

func TestEnsureNetherNetIdentityRejectsInvalidPEM(t *testing.T) {
	identityFile := filepath.Join(t.TempDir(), "identity.pem")
	if err := os.WriteFile(identityFile, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ensureNetherNetIdentity(identityFile); err == nil {
		t.Fatal("ensureNetherNetIdentity() succeeded with invalid PEM")
	}
}

func TestReadNetherNetIdentityAcceptsECPrivateKeyPEM(t *testing.T) {
	keyBytes, err := x509.MarshalECPrivateKey(mustGenerateIdentityKey(t))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "identity.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	}), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ensureNetherNetIdentity(path); err != nil {
		t.Fatalf("ensureNetherNetIdentity() rejected EC PRIVATE KEY PEM: %v", err)
	}
}

func mustGenerateIdentityKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}
