package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"sync"
)

const (
	defaultSecretKey = "daedalus-default-encryption-secret-key-32bytes!!"
)

var (
	once            sync.Once
	masterSecretKey []byte
)

func getMasterSecretKey() []byte {
	once.Do(func() {
		keyStr := os.Getenv("DAEDALUS_SECRET_ENCRYPTION_KEY")
		if keyStr == "" {
			keyStr = defaultSecretKey
		}
		hash := sha256.Sum256([]byte(keyStr))
		masterSecretKey = hash[:]
	})
	return masterSecretKey
}

// Encrypt encrypts plain text using AES-256-GCM.
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	key := getMasterSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts AES-256-GCM encrypted base64 text.
func Decrypt(encodedCipherText string) (string, error) {
	if encodedCipherText == "" {
		return "", nil
	}

	cipherText, err := base64.StdEncoding.DecodeString(encodedCipherText)
	if err != nil {
		// If it fails to decode base64, return original (in case it wasn't encrypted)
		return encodedCipherText, nil
	}

	key := getMasterSecretKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainTextBytes, err := gcm.Open(nil, nonce, actualCipherText, nil)
	if err != nil {
		// If decryption fails (e.g. key mismatch or plain string), return string as fallback
		return encodedCipherText, nil
	}

	return string(plainTextBytes), nil
}
