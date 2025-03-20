package sensitivevariables

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"strings"
)

func DecryptSensitiveVariable(masterKey string, value string) (string, error) {
	split := strings.Split(value, "|")

	if len(split) != 2 {
		return "", errors.New("expected two base64 encoded strings separated by a pipe")
	}

	cipherText, err := base64.StdEncoding.DecodeString(split[0])

	if err != nil {
		return "", err
	}

	iv, err := base64.StdEncoding.DecodeString(split[1])

	if err != nil {
		return "", err
	}

	key, err := base64.StdEncoding.DecodeString(masterKey)

	if err != nil {
		return "", err
	}

	// Create a new AES cipher block
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Check that the IV length is equal to the block size
	if len(iv) != aes.BlockSize {
		return "", errors.New("IV length must be equal to block size")
	}

	// Create a new CBC decrypter
	mode := cipher.NewCBCDecrypter(block, iv)

	// Decrypt the ciphertext
	decrypted := make([]byte, len(cipherText))
	mode.CryptBlocks(decrypted, cipherText)

	// Remove padding
	decrypted = PKCS7Unpad(decrypted)

	return string(decrypted), nil
}

func PKCS7Unpad(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}
