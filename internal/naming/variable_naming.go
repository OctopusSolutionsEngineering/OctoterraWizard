package naming

import "github.com/mcasperson/OctoterraWizard/internal/hash"

// VariableSecretName returns a unique name for the Terraform variable used to populate the
// Octopus sensitive variable. This name has to be unique to avoid conflicts and generated in
// a deterministic way to ensure that the same name is used when the export is run multiple times
// and also when the values are populated by external tools.
func VariableSecretName(id string) string {
	return hash.Sha256Hash(id) + "_sensitive_value"
}

func VariableValueName(id string) string {
	return hash.Sha256Hash(id) + "_value"
}
