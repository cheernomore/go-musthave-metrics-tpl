package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func genKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func writePEM(t *testing.T, path, blockType string, der []byte) {
	t.Helper()
	f, err := os.Create(path)
	require.NoError(t, err)
	defer f.Close()
	require.NoError(t, pem.Encode(f, &pem.Block{Type: blockType, Bytes: der}))
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := genKey(t)

	// Полезная нагрузка больше максимума RSA — проверяем гибридную схему.
	payload := make([]byte, 8192)
	_, err := rand.Read(payload)
	require.NoError(t, err)

	enc, err := Encrypt(&key.PublicKey, payload)
	require.NoError(t, err)
	assert.NotEqual(t, payload, enc)

	dec, err := Decrypt(key, enc)
	require.NoError(t, err)
	assert.Equal(t, payload, dec)
}

func TestDecrypt_WrongKey(t *testing.T) {
	enc, err := Encrypt(&genKey(t).PublicKey, []byte("secret"))
	require.NoError(t, err)

	_, err = Decrypt(genKey(t), enc)
	assert.Error(t, err)
}

func TestDecrypt_ShortData(t *testing.T) {
	_, err := Decrypt(genKey(t), []byte{0x00})
	assert.Error(t, err)
}

func TestLoadKeys_PKCS8AndPKIX(t *testing.T) {
	key := genKey(t)
	dir := t.TempDir()

	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	privPath := filepath.Join(dir, "private.pem")
	writePEM(t, privPath, "PRIVATE KEY", privDER)

	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPath := filepath.Join(dir, "public.pem")
	writePEM(t, pubPath, "PUBLIC KEY", pubDER)

	pub, err := LoadPublicKey(pubPath)
	require.NoError(t, err)
	priv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)

	enc, err := Encrypt(pub, []byte("hello"))
	require.NoError(t, err)
	dec, err := Decrypt(priv, enc)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(dec))
}

func TestLoadKeys_PKCS1(t *testing.T) {
	key := genKey(t)
	dir := t.TempDir()

	privPath := filepath.Join(dir, "private.pem")
	writePEM(t, privPath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(key))
	pubPath := filepath.Join(dir, "public.pem")
	writePEM(t, pubPath, "RSA PUBLIC KEY", x509.MarshalPKCS1PublicKey(&key.PublicKey))

	pub, err := LoadPublicKey(pubPath)
	require.NoError(t, err)
	priv, err := LoadPrivateKey(privPath)
	require.NoError(t, err)
	assert.Equal(t, key.N, pub.N)
	assert.Equal(t, key.D, priv.D)
}

func TestLoadKeys_Errors(t *testing.T) {
	_, err := LoadPublicKey(filepath.Join(t.TempDir(), "nope.pem"))
	assert.Error(t, err)
	_, err = LoadPrivateKey(filepath.Join(t.TempDir(), "nope.pem"))
	assert.Error(t, err)

	bad := filepath.Join(t.TempDir(), "bad.pem")
	require.NoError(t, os.WriteFile(bad, []byte("not a pem"), 0o644))
	_, err = LoadPublicKey(bad)
	assert.Error(t, err)
	_, err = LoadPrivateKey(bad)
	assert.Error(t, err)
}
