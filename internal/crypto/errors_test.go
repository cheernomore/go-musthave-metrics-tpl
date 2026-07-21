package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadKeys_NonRSA(t *testing.T) {
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	dir := t.TempDir()

	pubDER, err := x509.MarshalPKIXPublicKey(&ecKey.PublicKey)
	require.NoError(t, err)
	pubPath := filepath.Join(dir, "ec_pub.pem")
	writePEM(t, pubPath, "PUBLIC KEY", pubDER)

	_, err = LoadPublicKey(pubPath)
	assert.Error(t, err, "ECDSA-ключ не должен приниматься как RSA")

	privDER, err := x509.MarshalPKCS8PrivateKey(ecKey)
	require.NoError(t, err)
	privPath := filepath.Join(dir, "ec_priv.pem")
	writePEM(t, privPath, "PRIVATE KEY", privDER)

	_, err = LoadPrivateKey(privPath)
	assert.Error(t, err, "ECDSA-ключ не должен приниматься как RSA")
}

func TestDecrypt_TruncatedKey(t *testing.T) {
	// Заявлена длина ключа 256 байт, а данных меньше.
	data := []byte{0x01, 0x00, 0x01}
	_, err := Decrypt(genKey(t), data)
	assert.Error(t, err)
}

func TestDecrypt_ShortNonce(t *testing.T) {
	key := genKey(t)

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &key.PublicKey, make([]byte, 32), nil)
	require.NoError(t, err)

	data := binary.BigEndian.AppendUint16(nil, uint16(len(encKey)))
	data = append(data, encKey...)
	data = append(data, 1, 2, 3) // короче, чем nonce GCM

	_, err = Decrypt(key, data)
	assert.Error(t, err)
}
