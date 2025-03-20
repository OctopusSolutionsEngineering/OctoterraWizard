package main

import (
	"bytes"
	"encoding/json"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/spaces"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformTestFramework/octoclient"
	"github.com/OctopusSolutionsEngineering/OctopusTerraformTestFramework/test"
	"github.com/mcasperson/OctoterraWizard/internal/infrastructure"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/mcasperson/OctoterraWizard/internal/steps"
	"github.com/samber/lo"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestSpreadVariables(t *testing.T) {
	testFramework := test.OctopusContainerTest{}
	testFramework.ArrangeTest(t, func(t *testing.T, container *test.OctopusContainer, client *client.Client) error {
		// Act
		newSpaceId, err := testFramework.Act(
			t,
			container,
			filepath.Join("terraform"),
			"2-simpleexample",
			[]string{})

		if err != nil {
			return err
		}

		newSpaceClient, err := octoclient.CreateClient(container.URI, newSpaceId, test.ApiKey)

		if err != nil {
			return err
		}

		step := steps.SpreadVariablesStep{
			BaseStep: steps.BaseStep{State: state.State{
				BackendType:             "AWS S3",
				Server:                  container.URI,
				ApiKey:                  test.ApiKey,
				Space:                   newSpaceId,
				DestinationServer:       "",
				DestinationApiKey:       "",
				DestinationSpace:        "",
				AwsAccessKey:            "",
				AwsSecretKey:            "",
				AwsS3Bucket:             "",
				AwsS3BucketRegion:       "",
				PromptForDelete:         false,
				AzureResourceGroupName:  "",
				AzureStorageAccountName: "",
				AzureContainerName:      "",
				AzureSubscriptionId:     "",
				AzureTenantId:           "",
				AzureApplicationId:      "",
				AzurePassword:           "",
			}},
		}

		// we must be able to repeat this step with no changes
		for i := 0; i < 3; i++ {
			if err := step.Execute(); err != nil {
				t.Fatalf("Error executing step: %v", err)
			}

			lvs, err := newSpaceClient.LibraryVariableSets.GetAll()

			if err != nil {
				t.Fatalf("Error getting library variable sets: %v", err)
			}

			lvsVariable, err := variables.GetVariableSet(newSpaceClient, newSpaceClient.GetSpaceID(), lvs[0].VariableSetID)

			if err != nil {
				t.Fatalf("Error getting library variable sets: %v", err)
			}

			if len(lvsVariable.Variables) != 11 {
				t.Fatalf("Expected 11 variables, got %v", len(lvsVariable.Variables))
			}

			// There must be one regular variable that was unaltered
			if len(lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return !item.IsSensitive && item.Name == "RegularVariable"
			})) != 1 {
				t.Fatalf("Expected 1 regular variable")
			}

			// There must be one variable called "Test.SecretVariable_Unscoped"
			// This variable was created specifically to collide with the name of a newly created sensitive variable
			if len(lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "Test.SecretVariable_Unscoped" && !item.IsSensitive
			})) != 1 {
				t.Fatalf("Expected 1 regular variable")
			}

			// There must be one variable called "Test.SecretVariable_Unscoped_1"
			// This variable must have an index appended to avoid the collision with the variable above
			if len(lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "Test.SecretVariable_Unscoped_1" && item.IsSensitive
			})) != 1 {
				t.Fatalf("Expected 1 sensitive variable")
			}

			// There must be one variable with a value of  "#{Test.SecretVariable_Unscoped_1}"
			if len(lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.Value != nil && *item.Value == "#{Test.SecretVariable_Unscoped_1}" && !item.IsSensitive
			})) != 1 {
				t.Fatalf("Expected 1 regular variable referencing the sensitive variable with the index suffix")
			}

			// Only one spread variable (the variable above) should end with a digit
			clashedVariables := lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return regexp.MustCompile(`.*?_\d+$`).MatchString(item.Name)
			})

			if len(clashedVariables) != 1 {
				t.Fatalf("Expected 1 variable, got %v", len(clashedVariables))
			}

			// All sensitive variables must be unscoped
			if len(lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.IsSensitive && !item.Scope.IsEmpty()
			})) != 0 {
				t.Fatalf("Expected 0 sensitive variables to be unscoped")
			}

			// There must be 5 sensitive variables, and they must all be unscoped
			sensitiveVariables := lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.IsSensitive && item.Scope.IsEmpty()
			})

			if len(sensitiveVariables) != 5 {
				t.Fatalf("Expected 5 variables, got %v", len(sensitiveVariables))
			}

			// The four sensitive variables that shared a name must now have 4 regular variables each scoped
			// to an environment or are unscoped
			originalVariables := lo.Filter(lvsVariable.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "Test.SecretVariable" && !item.IsSensitive && (len(item.Scope.Environments) == 1 || len(item.Scope.Environments) == 0)
			})

			if len(originalVariables) != 4 {
				t.Fatalf("Expected 4 variables, got %v", len(originalVariables))
			}

			// Each regular variable must reference a sensitive variable
			for _, variable := range originalVariables {
				matchingSensitiveVar := lo.Filter(sensitiveVariables, func(item *variables.Variable, index int) bool {
					return *variable.Value == "#{"+item.Name+"}"
				})

				if len(matchingSensitiveVar) == 0 {
					t.Fatalf("Should have found a matching sensitive variable for %v", variable.Name)
				}
			}

			// No variables should end with a underscore
			if lo.ContainsBy(lvsVariable.Variables, func(item *variables.Variable) bool {
				return strings.HasSuffix(item.Name, "_")
			}) {
				t.Fatalf("No variables should end with an underscore")
			}

			// No variables should have whitespace around them
			if lo.ContainsBy(lvsVariable.Variables, func(item *variables.Variable) bool {
				return strings.TrimSpace(item.Name) != item.Name
			}) {
				t.Fatalf("No variables should have any whitespace around them")
			}

			// No sensitive variable should have the same name (this is the whole point of spreading)
			if lo.ContainsBy(lvsVariable.Variables, func(item *variables.Variable) bool {
				return lo.ContainsBy(lvsVariable.Variables, func(item2 *variables.Variable) bool {
					return item.IsSensitive && item2.IsSensitive && item != item2 && item.Name == item2.Name
				})
			}) {
				t.Fatalf("No sensitive variables should have the same name")
			}

			// No sensitive variable should have a scope (this is the whole point of spreading)
			if lo.ContainsBy(lvsVariable.Variables, func(item *variables.Variable) bool {
				return item.IsSensitive && !item.Scope.IsEmpty()
			}) {
				t.Fatalf("No sensitive variables should have a scope")
			}

		}

		return nil
	})
}

func TestProjectSpreadVariables(t *testing.T) {
	testFramework := test.OctopusContainerTest{}
	testFramework.ArrangeTest(t, func(t *testing.T, container *test.OctopusContainer, client *client.Client) error {
		// Act
		newSpaceId, err := testFramework.Act(
			t,
			container,
			filepath.Join("terraform"),
			"3-simpleproject",
			[]string{})

		if err != nil {
			return err
		}

		newSpaceClient, err := octoclient.CreateClient(container.URI, newSpaceId, test.ApiKey)

		if err != nil {
			return err
		}

		step := steps.SpreadVariablesStep{
			BaseStep: steps.BaseStep{State: state.State{
				BackendType:             "AWS S3",
				Server:                  container.URI,
				ApiKey:                  test.ApiKey,
				Space:                   newSpaceId,
				DestinationServer:       "",
				DestinationApiKey:       "",
				DestinationSpace:        "",
				AwsAccessKey:            "",
				AwsSecretKey:            "",
				AwsS3Bucket:             "",
				AwsS3BucketRegion:       "",
				PromptForDelete:         false,
				AzureResourceGroupName:  "",
				AzureStorageAccountName: "",
				AzureContainerName:      "",
				AzureSubscriptionId:     "",
				AzureTenantId:           "",
				AzureApplicationId:      "",
				AzurePassword:           "",
			}},
		}

		// we must be able to repeat this step with no changes
		for i := 0; i < 3; i++ {
			if err := step.Execute(); err != nil {
				t.Fatalf("Error executing step: %v", err)
			}

			project, err := projects.GetByName(newSpaceClient, newSpaceClient.GetSpaceID(), "Test")

			if err != nil {
				t.Fatalf("Error getting library project: %v", err)
			}

			variableSet, err := variables.GetVariableSet(newSpaceClient, newSpaceClient.GetSpaceID(), project.VariableSetID)

			if err != nil {
				t.Fatalf("Error getting project variable set: %v", err)
			}

			if len(variableSet.Variables) != 21 {
				t.Fatalf("Expected 21 variables, got %v", len(variableSet.Variables))
			}

			// The AWS account variable must be unaltered
			if len(lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "AWS" && item.Type == "AmazonWebServicesAccount"
			})) != 1 {
				t.Fatalf("Expected 1 AWS account variable")
			}

			// The regular variable that shares the name with the sensitive variables must be unchanged
			if len(lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "SensitiveVariable" && item.Type == "String" && *item.Value == "RegularVariable"
			})) != 1 {
				t.Fatalf("Expected 1 unchanged regular variable")
			}

			// All sensitive variables must be unscoped
			if len(lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				return item.IsSensitive && !item.Scope.IsEmpty()
			})) != 0 {
				t.Fatalf("Expected 0 sensitive variables to be unscoped")
			}

			// There must be 9 sensitive variables, and they must all be unscoped
			// One sensitive variable was already unscoped, the others were originally scoped and then spread
			// There are 10 variables called "SensitiveVariable", but only 9 are actually sensitive. One is regular variable thrown into the mix.
			sensitiveVariables := lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				return item.IsSensitive && item.Scope.IsEmpty()
			})

			if len(sensitiveVariables) != 9 {
				t.Fatalf("Expected 9 variables, got %v", len(sensitiveVariables))
			}

			// The 10 sensitive variables that shared a name must now have 9 regular variables each scoped
			// to an environment or are unscoped
			originalVariables := lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				// we're not testing the regular variable with the value "RegularVariable", as this was never modified
				return item.Name == "SensitiveVariable" && !item.IsSensitive && (len(item.Scope.Environments) == 1 || len(item.Scope.Environments) == 0) && *item.Value != "RegularVariable"
			})

			if len(originalVariables) != 9 {
				t.Fatalf("Expected 9 variables, got %v", len(originalVariables))
			}

			// Each regular variable must reference a sensitive variable
			for _, variable := range originalVariables {
				matchingSensitiveVar := lo.Filter(sensitiveVariables, func(item *variables.Variable, index int) bool {
					return *variable.Value == "#{"+item.Name+"}"
				})

				if len(matchingSensitiveVar) == 0 {
					t.Fatalf("Should have found a matching sensitive variable for %v", variable.Name)
				}
			}

			// There must be a regular variable that was not altered
			regularVariable := lo.Filter(variableSet.Variables, func(item *variables.Variable, index int) bool {
				return item.Name == "RegularVariable" && !item.IsSensitive
			})

			if len(regularVariable) != 1 {
				t.Fatalf("Expected 1 variable, got %v", len(regularVariable))
			}

			// No variables should end with an underscore
			if lo.ContainsBy(variableSet.Variables, func(item *variables.Variable) bool {
				return strings.HasSuffix(item.Name, "_")
			}) {
				t.Fatalf("No variables should end with an underscore")
			}

			// No variables should have whitespace around them
			if lo.ContainsBy(variableSet.Variables, func(item *variables.Variable) bool {
				return strings.TrimSpace(item.Name) != item.Name
			}) {
				t.Fatalf("No variables should have any whitespace around them")
			}

			// No sensitive variable should have the same name (this is the whole point of spreading)
			if lo.ContainsBy(variableSet.Variables, func(item *variables.Variable) bool {
				return lo.ContainsBy(variableSet.Variables, func(item2 *variables.Variable) bool {
					return item.IsSensitive && item2.IsSensitive && item != item2 && item.Name == item2.Name
				})
			}) {
				t.Fatalf("No sensitive variables should have the same name")
			}

			// No sensitive variable should have a scope (this is the whole point of spreading)
			if lo.ContainsBy(variableSet.Variables, func(item *variables.Variable) bool {
				return item.IsSensitive && !item.Scope.IsEmpty()
			}) {
				t.Fatalf("No sensitive variables should have a scope")
			}
		}

		return nil
	})
}

// TestProjectMigration tests a full space migration running the wizard steps in the same sequence
// a user would from the UI
func TestProjectMigration(t *testing.T) {
	testFramework := test.OctopusContainerTest{}
	testFramework.ArrangeTest(t, func(t *testing.T, container *test.OctopusContainer, client *client.Client) error {
		// Act
		newSpaceId, err := testFramework.Act(
			t,
			container,
			filepath.Join("terraform"),
			"3-simpleproject",
			[]string{})

		if err != nil {
			return err
		}

		newSpaceClient, err := octoclient.CreateClient(container.URI, newSpaceId, test.ApiKey)

		newSpace := spaces.NewSpace("Migration")
		newSpace.SpaceManagersTeams = []string{"teams-administrators"}
		space, err := newSpaceClient.Spaces.Add(newSpace)

		if err != nil {
			return err
		}

		state := state.State{
			BackendType:                   "AWS S3",
			Server:                        "http://localhost:8080", // The address used by Octopus when running tasks, which could be in nested containers
			ServerExternal:                container.URI,           // The address used by the wizard
			ApiKey:                        test.ApiKey,
			Space:                         newSpaceId,
			DestinationServer:             "http://localhost:8080",
			DestinationServerExternal:     container.URI,
			DestinationApiKey:             test.ApiKey,
			DestinationSpace:              space.ID,
			AwsAccessKey:                  os.Getenv("AWS_ACCESS_KEY_ID"),
			AwsSecretKey:                  os.Getenv("AWS_SECRET_ACCESS_KEY"),
			AwsS3Bucket:                   os.Getenv("AWS_DEFAULT_BUCKET"),
			AwsS3BucketRegion:             os.Getenv("AWS_DEFAULT_REGION"),
			PromptForDelete:               false,
			UseContainerImages:            false,
			AzureResourceGroupName:        "",
			AzureStorageAccountName:       "",
			AzureContainerName:            "",
			AzureSubscriptionId:           "",
			AzureTenantId:                 "",
			AzureApplicationId:            "",
			AzurePassword:                 "",
			ExcludeAllLibraryVariableSets: false,
			EnableVariableSpreading:       false,
			// The test framework needs to be set up to expose the database and define a known master key.
			// This is not done yet.
			// Once the test framework is set up to expose the database, the following values can be used:
			DatabaseServer:    "localhost",
			DatabaseUser:      "SA",
			DatabasePass:      "Password01!",
			DatabasePort:      "1433",
			DatabaseName:      "Octopus",
			DatabaseMasterKey: "",
		}

		// need to install pip and terraform onto the Octopus container
		installPip := "{\"Name\":\"AdHocScript\",\"Description\":\"Script run from management console\",\"Arguments\":{\"MachineIds\":[],\"TenantIds\":[],\"TargetRoles\":[],\"EnvironmentIds\":[],\"WorkerIds\":[],\"WorkerPoolIds\":[],\"TargetType\":\"OctopusServer\",\"Syntax\":\"Bash\",\"ScriptBody\":\"apt-get update && apt-get install -y gnupg software-properties-common\\nwget -O- https://apt.releases.hashicorp.com/gpg | \\\\\\ngpg --dearmor | \\\\\\ntee /usr/share/keyrings/hashicorp-archive-keyring.gpg > /dev/null\\necho \\\"deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] \\\\\\nhttps://apt.releases.hashicorp.com $(lsb_release -cs) main\\\" | \\\\\\ntee /etc/apt/sources.list.d/hashicorp.list\\napt update\\napt-get install -y terraform\\n\\ncurl -L -o /usr/bin/octoterra https://github.com/OctopusSolutionsEngineering/OctopusTerraformExport/releases/latest/download/octoterra_linux_amd64\\nchmod +x /usr/bin/octoterra\\n\\ncurl -L -o octopustools.tar.gz https://download.octopusdeploy.com/octopus-tools/9.0.0/OctopusTools.9.0.0.linux-x64.tar.gz\\ntar -xzf octopustools.tar.gz\\nchmod +x octo\\nmv octo /usr/bin/octo\\n\\napt-get update\\napt-get upgrade -y\\napt install python3-pip -y\"},\"SpaceId\":\"Spaces-1\"}"
		req, err := http.NewRequest("POST", container.URI+"/api/tasks", bytes.NewReader([]byte(installPip)))

		if err != nil {
			return err
		}

		response, err := newSpaceClient.HttpSession().DoRawRequest(req)

		if err != nil {
			return err
		}

		responseBody, err := io.ReadAll(response.Body)

		if err != nil {
			return err
		}

		stepTemplates := map[string]any{}
		if err := json.Unmarshal(responseBody, &stepTemplates); err != nil {
			return err
		}

		err = infrastructure.WaitForTask(state, stepTemplates["Id"].(string), func(message string) {})

		if err != nil {
			return err
		}

		// This can be enabled once the test framework is set up to expose the database
		//if err := (steps.ExtractSecrets{BaseStep: steps.BaseStep{State: state}}).Execute(); err != nil {
		//	Fatal(t, "Error executing ExtractSecrets: %v", err)
		//}

		if _, err := (steps.StepTemplateStep{BaseStep: steps.BaseStep{State: state}}).Execute(); err != nil {
			Fatal(t, "Error executing StepTemplateStep: %v", err)
		}

		steps.SpaceExportStep{BaseStep: steps.BaseStep{State: state}}.Execute(func(message string, body string, callback func(bool)) {
			callback(true)
		}, func(s string, err error) {
			Fatal(t, "Error executing SpaceExportStep: %v", err)
		}, func(s string) {
			// success
		}, false, false)

		steps.ProjectExportStep{BaseStep: steps.BaseStep{State: state}}.Execute(func(message string, body string, callback func(bool)) {
			callback(true)
		}, func(s string, err error) {
			Fatal(t, "Error executing ProjectExportStep: %v", err)
		}, func(s string) {
			// success
		}, func(s string) {
			// status
		})

		if err := (steps.StartSpaceExportStep{BaseStep: steps.BaseStep{State: state}}).Execute(func(message string) {}); err != nil {
			Fatal(t, "Error executing StartSpaceExportStep: %v", err)
		}

		if err := (steps.StartProjectExportStep{BaseStep: steps.BaseStep{State: state}}).Execute(func(message string) {}); err != nil {
			Fatal(t, "Error executing StartProjectExportStep: %v", err)
		}

		migratedSpaceClient, err := octoclient.CreateClient(container.URI, space.ID, test.ApiKey)

		if err != nil {
			return err
		}

		project, err := projects.GetByName(migratedSpaceClient, migratedSpaceClient.GetSpaceID(), "Test")

		if err != nil {
			return err
		}

		if project.Description != "Test project" {
			t.Fatalf("Expected description to be 'Test project', got %v", project.Description)
		}

		return nil
	})
}

func Fatal(t *testing.T, message string, err error) {
	if uw, ok := err.(interface{ Unwrap() []error }); ok {
		for _, e := range uw.Unwrap() {
			t.Fatalf(message, e)
		}
	} else {
		t.Fatalf(message, err)
	}
}
