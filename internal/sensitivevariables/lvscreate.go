package sensitivevariables

import (
	"errors"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/libraryvariablesets"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/samber/lo"
)

const SecretsLibraryVariableSetName = "SpaceSensitiveVars"
const SecretsVariableName = "OctoterraWiz.Terraform.Vars"

// CreateSecretsLibraryVariableSet will create or reuse a library variable set called OctoterraWizSecrets
// that containers a single sensitive variable called OctoterraWiz.Terraform.Vars. The variable is the contents
// of a terraform.tfvars file that contains all the sensitive variables required to migrate a space.
func CreateSecretsLibraryVariableSet(values string, state state.State) error {
	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return errors.Join(errors.New("failed to create client"), err)
	}

	// Find an existing library variable set or create a new one
	existingLvs, err := libraryvariablesets.Get(myclient, myclient.GetSpaceID(), variables.LibraryVariablesQuery{
		ContentType: "",
		IDs:         nil,
		PartialName: SecretsLibraryVariableSetName,
		Skip:        0,
		Take:        0,
	})

	if err != nil {
		return errors.Join(errors.New("failed to get library variable set"), err)
	}

	matchingLvs := lo.Filter(existingLvs.Items, func(item *variables.LibraryVariableSet, index int) bool {
		return item.Name == SecretsLibraryVariableSetName
	})

	var lvs *variables.LibraryVariableSet

	if len(matchingLvs) > 0 {
		lvs = matchingLvs[0]
	} else {
		lvs, err = libraryvariablesets.Add(myclient, variables.NewLibraryVariableSet(SecretsLibraryVariableSetName))

		if err != nil {
			return errors.Join(errors.New("failed to create library variable set"), err)
		}
	}

	// Delete any existing variable with the same name
	existingVariables, err := variables.GetAll(myclient, myclient.GetSpaceID(), lvs.ID)

	if err != nil {
		return errors.Join(errors.New("failed to get variables"), err)
	}

	for _, variable := range existingVariables.Variables {
		if variable.Name == SecretsVariableName {
			_, err = variables.DeleteSingle(myclient, myclient.GetSpaceID(), lvs.ID, variable.ID)

			if err != nil {
				return errors.Join(errors.New("failed to delete variable"), err)
			}
		}
	}

	// Create a new variable
	variable := variables.NewVariable(SecretsVariableName)
	variable.IsSensitive = true
	variable.Type = "Sensitive"
	variable.Value = &values

	_, err = variables.AddSingle(myclient, myclient.GetSpaceID(), lvs.ID, variable)

	if err != nil {
		return errors.Join(errors.New("failed to add variable"), err)
	}

	return nil
}
