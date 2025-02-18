package spreadvariables

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/channels"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/deployments"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/environments"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/machines"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/runbooks"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/samber/lo"
	"slices"
	"strings"
)

type OwnerVariablePair struct {
	OwnerID     string
	VariableSet *variables.VariableSet
}

type VariableSpreader struct {
	State  state.State
	cache  map[string]map[string]string
	client *client.Client
}

func (c *VariableSpreader) findSecretVariablesWithSharedName(variableSet *variables.VariableSet) ([]string, error) {
	groupedVariables := []string{}
	for _, variable := range variableSet.Variables {

		// The variable has to be sensitive
		if !variable.IsSensitive {
			continue
		}

		if variable.Type != "Sensitive" {
			continue
		}

		// If it has an empty scope but no other variables share the name we can leave this variable unprocessed
		if variable.Scope.IsEmpty() && !lo.ContainsBy(variableSet.Variables, func(item *variables.Variable) bool {
			return item.Name == variable.Name && item.ID != variable.ID
		}) {
			continue
		}

		if slices.Index(groupedVariables, variable.Name) == -1 {
			groupedVariables = append(groupedVariables, variable.Name)
		}
	}

	return groupedVariables, nil
}

// populateCache returns names for all the IDs used as variable scopes.
func (c *VariableSpreader) populateCache(variable *variables.Variable, parent *variables.VariableSet) error {
	c.cache = map[string]map[string]string{}
	c.cache["Environments"] = map[string]string{}
	c.cache["Machines"] = map[string]string{}
	c.cache["Channels"] = map[string]string{}
	c.cache["ProcessOwners"] = map[string]string{}
	c.cache["Actions"] = map[string]string{}

	for _, resourceId := range variable.Scope.Environments {
		if _, ok := c.cache["Environments"][resourceId]; ok {
			continue
		}

		if resource, err := environments.GetByID(c.client, c.client.GetSpaceID(), resourceId); err != nil {
			return err
		} else {
			c.cache["Environments"][resourceId] = resource.Name
		}
	}

	for _, resourceId := range variable.Scope.Machines {
		if _, ok := c.cache["Machines"][resourceId]; ok {
			continue
		}

		if resource, err := machines.GetByID(c.client, c.client.GetSpaceID(), resourceId); err != nil {
			return err
		} else {
			c.cache["Machines"][resourceId] = resource.Name
		}
	}

	for _, resourceId := range variable.Scope.Channels {
		if _, ok := c.cache["Channels"][resourceId]; ok {
			continue
		}

		if resource, err := channels.GetByID(c.client, c.client.GetSpaceID(), resourceId); err != nil {
			return err
		} else {
			c.cache["Channels"][resourceId] = resource.Name
		}
	}

	for _, resourceId := range variable.Scope.Actions {
		if _, ok := c.cache["Actions"][resourceId]; ok {
			continue
		}

		project, err := projects.GetByID(c.client, c.client.GetSpaceID(), parent.OwnerID)

		if err != nil {
			return err
		}

		deploymentProcess, err := deployments.GetDeploymentProcessByID(c.client, c.client.GetSpaceID(), project.DeploymentProcessID)

		if err != nil {
			return err
		}

		actions := lo.FlatMap(deploymentProcess.Steps, func(item *deployments.DeploymentStep, index int) []*deployments.DeploymentAction {
			return item.Actions
		})

		action := lo.Filter(actions, func(item *deployments.DeploymentAction, index int) bool {
			return item.ID == resourceId
		})

		if len(action) == 0 {
			return errors.New("Could not find action " + resourceId)
		}

		c.cache["Actions"][resourceId] = action[0].Name
	}

	for _, resourceId := range variable.Scope.ProcessOwners {
		if _, ok := c.cache["ProcessOwners"][resourceId]; ok {
			continue
		}

		if strings.HasPrefix(resourceId, "Runbooks-") {
			if resource, err := runbooks.GetByID(c.client, c.client.GetSpaceID(), resourceId); err != nil {
				return err
			} else {
				c.cache["ProcessOwners"][resourceId] = resource.Name
			}
		} else {
			if resource, err := projects.GetByID(c.client, c.client.GetSpaceID(), resourceId); err != nil {
				return err
			} else {
				c.cache["ProcessOwners"][resourceId] = resource.Name
			}
		}
	}

	return nil
}

func (c *VariableSpreader) buildUniqueVariableName(variable *variables.Variable, usedNamed []string) (string, error) {
	name := variable.Name

	if variable.Scope.IsEmpty() {
		name += "_Unscoped"
	}

	var namingErrors error = nil

	if len(variable.Scope.Environments) > 0 {
		resourceNames := lo.Map(variable.Scope.Environments, func(item string, index int) string {
			if resource, ok := c.cache["Environments"][item]; ok && len(strings.TrimSpace(resource)) != 0 {
				return resource
			}

			namingErrors = errors.Join(namingErrors, errors.New(fmt.Sprintf("Environment with ID %s not found", item)))
			return ""
		})
		name += fmt.Sprintf("_%s", strings.Join(resourceNames, "_"))
	}

	if len(variable.Scope.Machines) > 0 {
		resourceNames := lo.Map(variable.Scope.Machines, func(item string, index int) string {
			if resource, ok := c.cache["Machines"][item]; ok && len(strings.TrimSpace(resource)) != 0 {
				return resource
			}

			namingErrors = errors.Join(namingErrors, errors.New(fmt.Sprintf("Machine with ID %s not found", item)))
			return ""
		})
		name += fmt.Sprintf("_%s", strings.Join(resourceNames, "_"))
	}

	if len(variable.Scope.Roles) > 0 {
		name += fmt.Sprintf("_%s", strings.Join(variable.Scope.Roles, "_"))
	}

	if len(variable.Scope.Actions) > 0 {
		resourceNames := lo.Map(variable.Scope.Actions, func(item string, index int) string {
			if resource, ok := c.cache["Actions"][item]; ok && len(strings.TrimSpace(resource)) != 0 {
				return resource
			}

			namingErrors = errors.Join(namingErrors, errors.New(fmt.Sprintf("Actions with ID %s not found", item)))
			return ""
		})
		name += fmt.Sprintf("_%s", strings.Join(resourceNames, "_"))
	}

	if len(variable.Scope.TenantTags) > 0 {
		name += fmt.Sprintf("_%s", strings.Join(variable.Scope.TenantTags, "_"))
	}

	if len(variable.Scope.Channels) > 0 {
		resourceNames := lo.Map(variable.Scope.Channels, func(item string, index int) string {
			if resource, ok := c.cache["Channels"][item]; ok && len(strings.TrimSpace(resource)) != 0 {
				return resource
			}

			namingErrors = errors.Join(namingErrors, errors.New(fmt.Sprintf("Channel with ID %s not found", item)))
			return ""
		})
		name += fmt.Sprintf("_%s", strings.Join(resourceNames, "_"))
	}

	if len(variable.Scope.ProcessOwners) > 0 {
		resourceNames := lo.Map(variable.Scope.ProcessOwners, func(item string, index int) string {
			if resource, ok := c.cache["ProcessOwners"][item]; ok && len(strings.TrimSpace(resource)) != 0 {
				return resource
			}

			namingErrors = errors.Join(namingErrors, errors.New(fmt.Sprintf("Process Owner with ID %s not found", item)))
			return ""
		})
		name += fmt.Sprintf("_%s", strings.Join(resourceNames, "_"))
	}

	if namingErrors != nil {
		return "", namingErrors
	}

	startingName := name
	index := 1
	for slices.Index(usedNamed, name) != -1 {
		name = startingName + "_" + fmt.Sprint(index)
		index++
	}

	return name, nil
}

func (c *VariableSpreader) spreadVariables(client *client.Client, ownerId string, variableSet *variables.VariableSet) error {
	groupedVariables, err := c.findSecretVariablesWithSharedName(variableSet)

	if err != nil {
		return err
	}

	// Get a list of all the existing variable names. We can't reuse any of these names.
	usedNames := lo.Uniq(lo.Map(variableSet.Variables, func(item *variables.Variable, index int) string {
		return item.Name
	}))

	for _, groupedVariable := range groupedVariables {
		for _, variable := range variableSet.Variables {
			if groupedVariable != variable.Name {
				continue
			}

			// You can have a mix of sensitive and non-sensitive variables with the same name
			// Skip any non-sensitive variables
			if !(variable.IsSensitive && variable.Type == "Sensitive") {
				continue
			}

			// Copy the original variable
			originalVar := *variable

			// Lookup things like environments
			if err := c.populateCache(variable, variableSet); err != nil {
				return err
			}

			// Get a unique name
			uniqueName, err := c.buildUniqueVariableName(variable, usedNames)
			if err != nil {
				return err
			}

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

			_, err = variables.AddSingle(client, client.GetSpaceID(), ownerId, &referenceVar)

			if err != nil {
				return err
			}

			// Update the original variable with the new name and no scopes
			originalName := variable.Name
			usedNames = append(usedNames, uniqueName)

			if variable.Value != nil {
				panic("The value of the variable must be nil here, otherwise we may be overriding sensitive values")
			}

			fmt.Println("Renaming " + originalName + " to " + uniqueName + " and removing scopes for " + ownerId)

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

			_, err = variables.UpdateSingle(client, client.GetSpaceID(), ownerId, variable)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *VariableSpreader) SpreadAllVariables() error {
	myclient, err := octoclient.CreateClient(c.State)

	if err != nil {
		return err
	}

	c.client = myclient

	libraryVariableSets, err := c.client.LibraryVariableSets.GetAll()

	if err != nil {
		return err
	}

	projects, err := c.client.Projects.GetAll()

	if err != nil {
		return err
	}

	variableSets := []OwnerVariablePair{}

	// If we are not exporting library variable sets, we don't need to process them
	if !c.State.ExcludeAllLibraryVariableSets {
		for _, libraryVariableSet := range libraryVariableSets {
			variableSet, err := variables.GetVariableSet(c.client, c.client.GetSpaceID(), libraryVariableSet.VariableSetID)

			if err != nil {
				return errors.New("Failed to get variable set for library variable set " + libraryVariableSet.Name + ". Error was \"" + err.Error() + "\"")
			}

			variableSets = append(variableSets, OwnerVariablePair{
				OwnerID:     libraryVariableSet.ID,
				VariableSet: variableSet,
			})
		}
	}

	for _, project := range projects {
		variableSet, err := variables.GetVariableSet(c.client, c.client.GetSpaceID(), project.VariableSetID)

		if err != nil {
			return errors.New("Failed to get variable set for project " + project.Name + ". Error was \"" + err.Error() + "\"")
		}

		variableSets = append(variableSets, OwnerVariablePair{
			OwnerID:     project.ID,
			VariableSet: variableSet,
		})
	}

	for _, variableSet := range variableSets {

		err = c.spreadVariables(c.client, variableSet.OwnerID, variableSet.VariableSet)

		if err != nil {
			return err
		}
	}

	return nil
}
