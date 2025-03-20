package sensitivevariables

import (
	"testing"
)

func TestDecryptSensitiveVariable(t *testing.T) {
	// Prepare test data
	masterKey := "6EdU6IWsCtMEwk0kPKflQQ=="
	value := "tHdE5KI9QVdsFSq6F6HeSA==|7oD+XzuTFF1uCQLXm8A3eg=="

	// Call the function to test
	decryptedValue, err := DecryptSensitiveVariable(masterKey, value)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	// Check the result
	if decryptedValue != "success" {
		t.Errorf("Expected %s, got %s", "success", decryptedValue)
	}
}
