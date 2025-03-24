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

func TestAccountSecretName(t *testing.T) {
	expected := "account_test_account"
	result := AccountSecretName("Test Account")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestAccountCertName(t *testing.T) {
	expected := "account_test_account_cert"
	result := AccountCertName("Test Account")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestTenantVariableSecretName(t *testing.T) {
	expected := "tenantvariable_6cc41d5ec590ab78cccecf81ef167d418c309a4598e8e45fef78039f7d9aa9fe_sensitive_value"
	result := TenantVariableSecretName("test-id")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestGitCredentialSecretName(t *testing.T) {
	expected := "gitcredential_test_name_sensitive_value"
	result := GitCredentialSecretName("test-name")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMachineSecretName(t *testing.T) {
	expected := "machine_test_machine_sensitive_value"
	result := MachineSecretName("Test Machine")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCertificateDataName(t *testing.T) {
	expected := "certificate_test_cert_data"
	result := CertificateDataName("Test Cert")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestCertificatePasswordName(t *testing.T) {
	expected := "certificate_test_cert_password"
	result := CertificatePasswordName("Test Cert")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFeedSecretName(t *testing.T) {
	expected := "feed_test_feed_password"
	result := FeedSecretName("Test Feed")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFeedSecretKeyName(t *testing.T) {
	expected := "feed_test_feed_secretkey"
	result := FeedSecretKeyName("Test Feed")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestStepPropertySecretName(t *testing.T) {
	parentId := "parent-123"
	actionId := "action-456"
	property := "password"
	expected := "action_32511561b47b9d170c206b50965262edbfdc56afcff4e23a591c8cb0d55317d9_sensitive_value"
	result := StepPropertySecretName(parentId, actionId, property)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestStepTemplateParameterSecretName(t *testing.T) {
	parentId := "template-123"
	parameterId := "param-456"
	expected := "steptemplate_87f373143ce71aaaa6db36039fd932e03c9293c3785c1a685575dde798924d88_sensitive_value"
	result := StepTemplateParameterSecretName(parentId, parameterId)
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestMachineProxyPassword(t *testing.T) {
	expected := "machine_proxy_test_machine_password"
	result := MachineProxyPassword("Test Machine")
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Test Name", "test_name"},
		{"test-name", "test_name"},
		{"123test", "_123test"},
		{"Test@Name#$%", "test_name___"},
		{"_TestName", "_testname"},
	}

	for _, tc := range tests {
		result := sanitizeName(tc.input)
		if result != tc.expected {
			t.Errorf("sanitizeName(%s): expected %s, got %s", tc.input, tc.expected, result)
		}
	}
}
