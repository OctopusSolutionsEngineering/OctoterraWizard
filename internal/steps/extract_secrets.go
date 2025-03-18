package steps

import (
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
)

// ExtractSecrets provides a step in the wizard to extract secrets from the Octopus database
type ExtractSecrets struct {
	BaseStep
	Wizard wizard.Wizard
}

// SaveSecretsVariable creates a library variable set with a secret value containing the contents
// of a terraform.tfvars file that populates the secrets used by the exported space
func (s *ExtractSecrets) SaveSecretsVariable() error {
	return nil
}
