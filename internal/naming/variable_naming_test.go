package naming

import (
	"testing"
)

func TestVariableSecretName(t *testing.T) {
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_sensitive_value" // Replace with the actual expected hash
	result := VariableSecretName("test-id")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestVariableValueName(t *testing.T) {
	expected := "variable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_value" // Replace with the actual expected hash
	result := VariableValueName("test-id")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
