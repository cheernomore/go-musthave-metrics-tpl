// Package crypto реализует асимметричное (гибридное) шифрование сообщений
// между агентом и сервером.
//
// Тело сообщения шифруется симметрично (AES-256-GCM), а случайный AES-ключ
// шифруется асимметрично (RSA-OAEP) публичным ключом получателя. Такой подход
// снимает ограничение RSA на размер шифруемых данных и позволяет передавать
// пакеты метрик произвольного размера.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// EncryptedHeader — HTTP-заголовок, которым агент помечает зашифрованное тело.
const EncryptedHeader = "X-Encrypted"

// LoadPublicKey читает и разбирает RSA-публичный ключ из PEM-файла.
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение публичного ключа: %w", err)
	}
	return parsePublicKey(data)
}

// LoadPrivateKey читает и разбирает RSA-приватный ключ из PEM-файла.
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение приватного ключа: %w", err)
	}
	return parsePrivateKey(data)
}

func parsePublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("crypto: не удалось декодировать PEM с публичным ключом")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		pub, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("crypto: публичный ключ не является RSA-ключом")
		}
		return pub, nil
	}
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto: разбор публичного ключа: %w", err)
	}
	return pub, nil
}

func parsePrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("crypto: не удалось декодировать PEM с приватным ключом")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	keyAny, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("crypto: разбор приватного ключа: %w", err)
	}
	priv, ok := keyAny.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("crypto: приватный ключ не является RSA-ключом")
	}
	return priv, nil
}

// Encrypt шифрует data публичным ключом pub. Формат результата:
//
//	[2 байта: длина зашифрованного AES-ключа][зашифрованный AES-ключ][nonce][шифртекст]
func Encrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	aesKey := make([]byte, 32) // AES-256
	if _, err := rand.Read(aesKey); err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
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
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	encKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: шифрование ключа: %w", err)
	}

	out := make([]byte, 0, 2+len(encKey)+len(nonce)+len(ciphertext))
	out = binary.BigEndian.AppendUint16(out, uint16(len(encKey)))
	out = append(out, encKey...)
	out = append(out, nonce...)
	out = append(out, ciphertext...)
	return out, nil
}

// Decrypt расшифровывает данные, полученные из Encrypt, приватным ключом priv.
func Decrypt(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, errors.New("crypto: слишком короткие данные")
	}
	keyLen := int(binary.BigEndian.Uint16(data[:2]))
	data = data[2:]
	if len(data) < keyLen {
		return nil, errors.New("crypto: повреждённые данные (ключ)")
	}
	encKey := data[:keyLen]
	data = data[keyLen:]

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, encKey, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: расшифровка ключа: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("crypto: повреждённые данные (nonce)")
	}
	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("crypto: расшифровка данных: %w", err)
	}
	return plaintext, nil
}
