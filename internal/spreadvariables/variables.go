package spreadvariables

import (
	"encoding/json"
	"fmt"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"slices"
)

func findSecretVariablesWithSharedNameAndScoped(variableSet *variables.VariableSet) ([]string, error) {
	groupedVariables := []string{}
	for _, variable := range variableSet.Variables {
		if !variable.IsSensitive {
			continue
		}

		if variable.Type != "Sensitive" {
			continue
		}

		if len(variable.Scope.Environments) == 0 &&
			len(variable.Scope.Machines) == 0 &&
			len(variable.Scope.Roles) == 0 &&
			len(variable.Scope.Actions) == 0 &&
			len(variable.Scope.TenantTags) == 0 &&
			len(variable.Scope.ProcessOwners) == 0 &&
			len(variable.Scope.Channels) == 0 {
			continue
		}

		groupedVariables = append(groupedVariables, variable.Name)
	}

	return groupedVariables, nil
}

func buildUniqueVariableName(variable *variables.Variable, usedNamed []string) string {
	name := variable.Name

	if len(variable.Scope.Environments) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.Environments[0])
	}

	if len(variable.Scope.Machines) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.Machines[0])
	}

	if len(variable.Scope.Roles) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.Roles[0])
	}

	if len(variable.Scope.Actions) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.Actions[0])
	}

	if len(variable.Scope.TenantTags) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.TenantTags[0])
	}

	if len(variable.Scope.Channels) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.Channels[0])
	}

	if len(variable.Scope.ProcessOwners) > 0 {
		name += fmt.Sprintf("_%s", variable.Scope.ProcessOwners[0])
	}

	startingName := name
	index := 1
	for slices.Index(usedNamed, name) != -1 {
		name = startingName + "_" + fmt.Sprint(index)
		index++
	}

	return name
}

func spreadVariables(client *client.Client, libraryVariableSet *variables.LibraryVariableSet, variableSet *variables.VariableSet) error {
	groupedVariables, err := findSecretVariablesWithSharedNameAndScoped(variableSet)

	if err != nil {
		return err
	}

	usedNames := []string{}
	for _, groupedVariable := range groupedVariables {
		for _, variable := range variableSet.Variables {
			if groupedVariable != variable.Name {
				continue
			}

			// Copy the original variable
			originalVar := *variable

			// Get a unique name
			uniqueName := buildUniqueVariableName(variable, usedNames)

			// Create a new variable with the original name and scopes referencing the new unscoped variable
			referenceVar := originalVar

			jsonData, err := json.Marshal(referenceVar.Scope)
			if err != nil {
				return err
			}

			// Note the original scope of this variable
			referenceVar.Description += "\n\nReplaced variable ID\n\n" + referenceVar.ID
			referenceVar.Description += "\n\nOriginal Scope\n\n" + string(jsonData)

			referenceVar.IsSensitive = false
			referenceVar.Type = "String"
			referenceVar.ID = ""
			reference := "#{" + uniqueName + "}"
			referenceVar.Value = &reference

			fmt.Println("Recreating " + referenceVar.Name + " referencing " + reference)

			_, err = variables.AddSingle(client, client.GetSpaceID(), libraryVariableSet.ID, &referenceVar)

			if err != nil {
				return err
			}

			// Update the original variable with the new name and no scopes
			originalName := variable.Name
			usedNames = append(usedNames, uniqueName)

			if variable.Value != nil {
				panic("The value of the variable must be nil here, otherwise we may be overriding sensitive values")
			}

			fmt.Println("Renaming " + originalName + " to " + uniqueName + " and removing scopes")

			jsonData, err = json.Marshal(variable.Scope)
			if err != nil {
				return err
			}

			// Note the original scope of this variable
			referenceVar.Description += "\n\nOriginal Name\n\n" + variable.Name
			// Note the original scope of this variable
			referenceVar.Description += "\n\nOriginal Scope\n\n" + string(jsonData)

			variable.Name = uniqueName
			variable.Scope = variables.VariableScope{}

			_, err = variables.UpdateSingle(client, client.GetSpaceID(), libraryVariableSet.ID, variable)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func SpreadAllVariables(state state.State) error {
	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return err
	}

	libraryVariableSets, err := myclient.LibraryVariableSets.GetAll()

	if err != nil {
		return err
	}

	for _, libraryVariableSet := range libraryVariableSets {
		variableSet, err := variables.GetVariableSet(myclient, myclient.GetSpaceID(), libraryVariableSet.VariableSetID)

		if err != nil {
			return err
		}

		err = spreadVariables(myclient, libraryVariableSet, variableSet)

		if err != nil {
			return err
		}
	}

	return nil
}