package naming

import (
	"github.com/mcasperson/OctoterraWizard/internal/hash"
	"regexp"
	"strings"
)

var allowedChars = regexp.MustCompile(`[^A-Za-z0-9]`)
var startsWithLetterOrUnderscore = regexp.MustCompile(`^[A-Za-z_].*`)

// VariableSecretName returns a unique name for the Terraform variable used to populate the
// Octopus sensitive variable. This name has to be unique to avoid conflicts and generated in
// a deterministic way to ensure that the same name is used when the export is run multiple times
// and also when the values are populated by external tools.
func VariableSecretName(id string) string {
	return "variable_" + hash.Sha256Hash(id) + "_sensitive_value"
}

func VariableValueName(id string) string {
	return "variable_" + hash.Sha256Hash(id) + "_value"
}

func AccountSecretName(name string) string {
	return "account_" + sanitizeName(name)
}

func AccountCertName(name string) string {
	return "account_" + sanitizeName(name) + "_cert"
}

func TenantVariableSecretName(id string) string {
	return "tenantvariable_" + hash.Sha256Hash(id) + "_sensitive_value"
}

func GitCredentialSecretName(id string) string {
	return "gitcredential_" + hash.Sha256Hash(id) + "_sensitive_value"
}

func MachineSecretName(name string) string {
	return "machine_" + sanitizeName(name) + "_sensitive_value"
}

func CertificateDataName(name string) string {
	return "certificate_" + sanitizeName(name) + "_data"
}

func CertificatePasswordName(name string) string {
	return "certificate_" + sanitizeName(name) + "_password"
}

func FeedSecretName(name string) string {
	return "feed_" + sanitizeName(name) + "_password"
}

func FeedSecretKeyName(name string) string {
	return "feed_" + sanitizeName(name) + "_secretkey"
}

// sanitizeName creates a string that can be used as a name for HCL resources
// From the Terraform docs:
// A name must start with a letter or underscore and may contain only letters, digits, underscores, and dashes.
func sanitizeName(name string) string {
	sanitized := allowedChars.ReplaceAllString(strings.ToLower(name), "_")
	if !startsWithLetterOrUnderscore.MatchString(sanitized) {
		return "_" + sanitized
	}
	return sanitized
}
