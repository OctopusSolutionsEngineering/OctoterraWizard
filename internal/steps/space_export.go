package steps

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/accounts"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/feeds"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/libraryvariablesets"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projectgroups"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/mcasperson/OctoterraWizard/internal/logutil"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/query"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"github.com/samber/lo"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed modules/space_management/terraform.tf
var module string

var spaceManagementProject = "Octoterra Space Management"

type LibraryVariableSetUsage struct {
	Projects []LibraryVariableSetUsageProjects `json:"Projects"`
}

type LibraryVariableSetUsageProjects struct {
	ProjectId string `json:"ProjectId"`
}

type SpaceExportStep struct {
	BaseStep
	Wizard        wizard.Wizard
	createProject *widget.Button
	infinite      *widget.ProgressBarInfinite
	result        *widget.Label
	logs          *widget.Entry
	next          *widget.Button
	previous      *widget.Button
	exportDone    bool
}

func (s SpaceExportStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, thisPrevious, thisNext := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(PromptRemovalStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		moveNext := func(proceed bool) {
			if !proceed {
				return
			}

			s.Wizard.ShowWizardStep(ProjectExportStep{
				Wizard:   s.Wizard,
				BaseStep: BaseStep{State: s.State}})
		}
		if !s.exportDone {
			dialog.NewConfirm(
				"Do you want to skip this step?",
				"If you have run this step previously you can skip this step", moveNext, s.Wizard.Window).Show()
		} else {
			moveNext(true)
		}
	})
	s.next = thisNext
	s.previous = thisPrevious
	s.exportDone = false

	heading := widget.NewLabel("Space Serialization Runbooks")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	intro := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		We now must create a project with runbooks to serialize the space to a Terraform module and reapply it to a new space.
		This project is called "Octoterra Space Management" in the project group "Octoterra".
		Click the "Create Project" button to create the project and its associated runbooks.
	`))
	s.infinite = widget.NewProgressBarInfinite()
	s.infinite.Start()
	s.infinite.Hide()
	s.result = widget.NewLabel("")
	s.logs = widget.NewEntry()
	s.logs.Disable()
	s.logs.MultiLine = true
	s.logs.Hide()
	s.logs.SetMinRowsVisible(20)
	s.createProject = widget.NewButton("Create Project", func() {
		s.exportDone = true
		s.createNewProject(parent)
	})
	middle := container.New(layout.NewVBoxLayout(), heading, intro, s.createProject, s.infinite, s.result, s.logs)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s SpaceExportStep) createNewProject(parent fyne.Window) {
	s.logs.SetText("")
	s.next.Disable()
	s.previous.Disable()
	s.infinite.Show()
	s.createProject.Disable()
	s.logs.Hide()
	s.result.SetText("🔵 Creating project. This can take a little while.")

	s.Execute(func(title string, message string, callback func(bool)) {
		dialog.NewConfirm(title, message, callback, parent).Show()
	}, func(title string, err error) {
		if err := logutil.WriteTextToFile("space_export_error.txt", err.Error()); err != nil {
			fmt.Println("Failed to write error to file")
		}

		s.result.SetText(title)
		s.logs.Show()
		s.logs.SetText(err.Error())
		s.infinite.Hide()
		s.previous.Enable()
	}, func(message string) {
		s.result.SetText(message)
		s.next.Enable()
		s.previous.Enable()
		s.infinite.Hide()
		s.logs.Hide()
	}, false, false)

}

func (s SpaceExportStep) Execute(prompt func(string, string, func(bool)), handleError func(string, error), handleSuccess func(string), attemptedLvsDelete bool, attemptedAccountDelete bool) {
	myclient, err := octoclient.CreateClient(s.State)

	if err != nil {
		handleError("🔴 Failed to create the client", err)
		return
	}

	// Best effort at deleting existing project and project group
	projExists, project, err := s.projectExists(myclient)

	if projExists {
		deleteProjectFunc := func(b bool) {
			if b {
				if err := s.deleteProject(myclient, project); err != nil {
					handleError("🔴 Failed to delete the resource", err)
				} else if s.State.PromptForDelete {
					s.Execute(prompt, handleError, handleSuccess, attemptedLvsDelete, attemptedAccountDelete)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Project Group Exists", "The project "+spaceManagementProject+" already exists. Do you want to delete it? It is usually safe to delete this resource.", deleteProjectFunc)
			// We can't go further until the group is deleted
			return
		} else {
			deleteProjectFunc(true)
		}
	}

	pgExists, pggroup, err := s.projectGroupExists(myclient)

	if pgExists {
		deleteProgGroupFunc := func(b bool) {
			if b {
				if err := s.deleteProjectGroup(myclient, pggroup); err != nil {
					handleError("🔴 Failed to delete the resource", err)
				} else if s.State.PromptForDelete {
					s.Execute(prompt, handleError, handleSuccess, attemptedLvsDelete, attemptedAccountDelete)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Project Group Exists", "The project group Octoterra already exists. Do you want to delete it? It is usually safe to delete this resource.", deleteProgGroupFunc)
			// We can't go further until the group is deleted
			return
		} else {
			deleteProgGroupFunc(true)
		}
	}

	lvsExists, lvs, err := query.LibraryVariableSetExists(myclient)

	if lvsExists && !attemptedLvsDelete {
		deleteLvsFunc := func(b bool) {
			if b {
				server := s.State.ServerExternal
				if server == "" {
					server = s.State.Server
				}

				// got to start by unlinking the project from all the projects
				var body io.Reader
				req, err := http.NewRequest("GET", server+"/api/"+s.State.Space+"/LibraryVariableSets/"+lvs.ID+"/usages", body)

				if err != nil {
					handleError("🔴 Failed to create the library variable set usage request", err)
					return
				}

				response, err := myclient.HttpSession().DoRawRequest(req)

				if err != nil {
					handleError("🔴 Failed to get the library variable set usage", err)
					return
				}

				responseBody, err := io.ReadAll(response.Body)

				if err != nil {
					handleError("🔴 Failed to read the library variable set query body", err)
					return
				}

				fmt.Print(string(responseBody))

				usage := LibraryVariableSetUsage{}
				if err := json.Unmarshal(responseBody, &usage); err != nil {
					handleError("🔴 Failed to unmarshal the library variable set usage response", err)
					return
				}

				if usage.Projects == nil {
					usage.Projects = []LibraryVariableSetUsageProjects{}
				}

				for _, projectReference := range usage.Projects {
					project, err := projects.GetByID(myclient, myclient.GetSpaceID(), projectReference.ProjectId)

					if err != nil {
						handleError("🔴 Failed to get project "+projectReference.ProjectId, err)
						return
					}

					project.IncludedLibraryVariableSets = lo.Filter(project.IncludedLibraryVariableSets, func(projectLvs string, index int) bool {
						return projectLvs != lvs.ID
					})

					_, err = projects.Update(myclient, project)

					if err != nil {
						handleError("🔴 Failed to update project "+projectReference.ProjectId, err)
						return
					}
				}

				// then we can delete the variable set
				if err := s.deleteLibraryVariableSet(myclient, lvs); err != nil {
					// we can't delete variable sets used in releases, but we can rename them
					if err := s.renameLibraryVariableSet(myclient, lvs); err != nil {
						handleError("🔴 Failed to rename library variable set "+lvs.Name, err)
						return
					}
				}

				if s.State.PromptForDelete {
					// Tolerate the inability to delete a LVS, because it might have been
					// captured in a release or runbook snapshot, which becomes very hard
					// to unwind.
					s.Execute(prompt, handleError, handleSuccess, true, attemptedAccountDelete)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Library Variable Set Exists", "The library variable set Octoterra already exists. Do you want to unlink it from all the projects and delete it? It is usually safe to delete this resource.", deleteLvsFunc)
			// We can't go further until the group is deleted
			return
		} else {
			deleteLvsFunc(true)
		}

	}

	feedExists, feed, err := s.feedExists(myclient)

	if feedExists {
		delteFeedFunc := func(b bool) {
			if b {
				if err := s.deleteFeed(myclient, feed); err != nil {
					handleError("🔴 Failed to delete the resource", err)
				} else if s.State.PromptForDelete {
					s.Execute(prompt, handleError, handleSuccess, attemptedLvsDelete, attemptedAccountDelete)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Feed Exists", "The feed Octoterra Docker Feed already exists. Do you want to delete it? It is usually safe to delete this resource.", delteFeedFunc)
			// We can't go further until the feed is deleted
			return
		} else {
			delteFeedFunc(true)
		}
	}

	awsAccountExists, awsAccount, err := s.accountExists(myclient, "Octoterra AWS Account")

	if awsAccountExists && !attemptedAccountDelete {
		deleteAccountFunc := func(b bool) {
			if b {
				// accounts can not be deleted if they are used by library variable sets
				if err := s.deleteAccount(myclient, awsAccount); err != nil {
					// we can rename accounts though
					if err := s.renameAccount(myclient, awsAccount); err != nil {
						fmt.Println(err.Error())
					}
				}

				if s.State.PromptForDelete {
					s.Execute(prompt, handleError, handleSuccess, attemptedLvsDelete, true)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Account Exists", "The account Octoterra AWS Account already exists. Do you want to delete it? It is usually safe to delete this resource.", deleteAccountFunc)
			// We can't go further until the account is deleted
			return
		} else {
			deleteAccountFunc(true)
		}
	}

	azureAccountExists, azureAccount, err := s.accountExists(myclient, "Octoterra Azure Account")

	if azureAccountExists && !attemptedAccountDelete {
		deleteAccountFunc := func(b bool) {
			if b {
				if err := s.deleteAccount(myclient, azureAccount); err != nil {
					if err := s.renameAccount(myclient, azureAccount); err != nil {
						fmt.Println(err.Error())
					}
				}

				if s.State.PromptForDelete {
					s.Execute(prompt, handleError, handleSuccess, attemptedLvsDelete, true)
				}
			}
		}

		if s.State.PromptForDelete {
			prompt("Account Exists", "The account Octoterra Azure Account already exists. Do you want to delete it? It is usually safe to delete this resource.", deleteAccountFunc)
			// We can't go further until the account is deleted
			return
		} else {
			deleteAccountFunc(true)
		}
	}

	// Find the step template ID
	serializeSpaceTemplate, err, message := query.GetStepTemplateId(myclient, s.State, "Octopus - Serialize Space to Terraform")

	if err != nil {
		handleError(message, err)
	}

	deploySpaceTemplateS3, err, message := query.GetStepTemplateId(myclient, s.State, "Octopus - Populate Octoterra Space (S3 Backend)")

	if err != nil {
		handleError(message, err)
	}

	deploySpaceTemplateAzureStorage, err, message := query.GetStepTemplateId(myclient, s.State, "Octopus - Populate Octoterra Space (Azure Backend)")

	if err != nil {
		handleError(message, err)
	}

	// Find space name
	spaceName, err := query.GetSpaceName(myclient, s.State)

	if err != nil {
		handleError(message, err)
	}

	// Save and apply the module
	dir, err := ioutil.TempDir("", "octoterra")
	if err != nil {
		handleError("🔴 An error occurred while creating a temporary directory", err)
	}

	filePath := filepath.Join(dir, "terraform.tf")
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			// ignore this and move on
			fmt.Println(err.Error())
		}
	}(filePath)

	if err := os.WriteFile(filePath, []byte(module), 0644); err != nil {
		handleError("🔴 An error occurred while writing the Terraform file", err)
	}

	initCmd := exec.Command("terraform", "init", "-no-color")
	initCmd.Dir = dir

	var initStdout, initStderr bytes.Buffer
	initCmd.Stdout = &initStdout
	initCmd.Stderr = &initStderr

	if err := initCmd.Run(); err != nil {
		handleError("🔴 Terraform init failed.", errors.New(err.Error()+"\n"+initStdout.String()+initCmd.String()))
	}

	applyCmd := exec.Command("terraform",
		"apply",
		"-auto-approve",
		"-no-color",
		"-var=octopus_serialize_actiontemplateid="+serializeSpaceTemplate,
		"-var=octopus_deploys3_actiontemplateid="+deploySpaceTemplateS3,
		"-var=octopus_deployazure_actiontemplateid="+deploySpaceTemplateAzureStorage,
		"-var=terraform_backend="+s.State.BackendType,
		"-var=use_container_images="+fmt.Sprint(s.State.UseContainerImages),
		"-var=octopus_server_external="+s.State.GetExternalServer(),
		"-var=octopus_server="+s.State.Server,
		"-var=octopus_apikey="+s.State.ApiKey,
		"-var=octopus_space_id="+s.State.Space,
		"-var=octopus_space_name="+spaceName,
		"-var=terraform_state_bucket="+s.State.AwsS3Bucket,
		"-var=terraform_state_bucket_region="+s.State.AwsS3BucketRegion,
		"-var=terraform_state_aws_accesskey="+s.State.AwsAccessKey,
		"-var=terraform_state_aws_secretkey="+s.State.AwsSecretKey,
		"-var=terraform_state_azure_resource_group="+s.State.AzureResourceGroupName,
		"-var=terraform_state_azure_storage_account="+s.State.AzureStorageAccountName,
		"-var=terraform_state_azure_storage_container="+s.State.AzureContainerName,
		"-var=terraform_state_azure_application_id="+s.State.AzureApplicationId,
		"-var=terraform_state_azure_subscription_id="+s.State.AzureSubscriptionId,
		"-var=terraform_state_azure_tenant_id="+s.State.AzureTenantId,
		"-var=terraform_state_azure_password="+s.State.AzurePassword,
		"-var=octopus_destination_server="+s.State.DestinationServer,
		"-var=octopus_destination_apikey="+s.State.DestinationApiKey,
		"-var=ignore_all_library_variable_sets="+fmt.Sprint(s.State.ExcludeAllLibraryVariableSets),
		"-var=octopus_destination_space_id="+s.State.DestinationSpace)
	applyCmd.Dir = dir

	var stdout, stderr bytes.Buffer
	applyCmd.Stdout = &stdout
	applyCmd.Stderr = &stderr

	if err := applyCmd.Run(); err != nil {
		handleError("🔴 Terraform apply failed", errors.New(stdout.String()+stderr.String()))
	} else {
		handleSuccess("🟢 Terraform apply succeeded")
		fmt.Println(stdout.String() + stderr.String())
	}
}

func (s SpaceExportStep) deleteProjectGroup(myclient *client.Client, projectGroup *projectgroups.ProjectGroup) error {
	if err := myclient.ProjectGroups.DeleteByID(projectGroup.ID); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) deleteProject(myclient *client.Client, project *projects.Project) error {
	fmt.Println("Attempting to delete project " + fmt.Sprint(project.ID))
	if err := myclient.Projects.DeleteByID(project.ID); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) deleteFeed(myclient *client.Client, feed feeds.IFeed) error {
	fmt.Println("Attempting to delete feed " + fmt.Sprint(feed.GetID()))
	if err := myclient.Feeds.DeleteByID(feed.GetID()); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) deleteAccount(myclient *client.Client, account accounts.IAccount) error {
	fmt.Println("Attempting to delete account " + fmt.Sprint(account.GetID()))
	if err := myclient.Accounts.DeleteByID(account.GetID()); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) renameAccount(myclient *client.Client, account accounts.IAccount) error {
	index := 1

	for {
		name := account.GetName() + " (old " + fmt.Sprint(index) + ")"
		allAccounts, err := accounts.GetAll(myclient, myclient.GetSpaceID())

		if err != nil {
			return err
		}

		exactMatches := lo.Filter(allAccounts, func(account accounts.IAccount, index int) bool {
			return account.GetName() == name
		})

		if len(exactMatches) == 0 {
			break
		}

		index++
	}

	fmt.Println("Attempting to rename account " + fmt.Sprint(account.GetID()))

	account.SetName(account.GetName() + " (old " + fmt.Sprint(index) + ")")
	if _, err := accounts.Update(myclient, account); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) projectExists(myclient *client.Client) (bool, *projects.Project, error) {
	if project, err := projects.GetByName(myclient, myclient.GetSpaceID(), spaceManagementProject); err == nil {
		return true, project, nil
	} else {
		return false, nil, err
	}
}

func (s SpaceExportStep) projectGroupExists(myclient *client.Client) (bool, *projectgroups.ProjectGroup, error) {
	if projectGroups, err := projectgroups.GetAll(myclient, myclient.GetSpaceID()); err == nil {
		groups := lo.Filter(projectGroups, func(pg *projectgroups.ProjectGroup, index int) bool {
			return pg.Name == "Octoterra"
		})

		if len(groups) == 0 {
			return false, nil, nil
		}

		return true, groups[0], nil
	} else {
		return false, nil, err
	}
}

func (s SpaceExportStep) deleteLibraryVariableSet(myclient *client.Client, lvs *variables.LibraryVariableSet) error {
	fmt.Println("Attempting to delete lvs " + fmt.Sprint(lvs.GetID()))
	if err := myclient.LibraryVariableSets.DeleteByID(lvs.ID); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) renameLibraryVariableSet(myclient *client.Client, lvs *variables.LibraryVariableSet) error {

	index := 1

	for {
		name := lvs.Name + " (old " + fmt.Sprint(index) + ")"
		existingLvs, err := myclient.LibraryVariableSets.GetByPartialName(name)

		if err != nil {
			return err
		}

		exactMatches := lo.Filter(existingLvs, func(lvs *variables.LibraryVariableSet, index int) bool {
			return lvs.Name == name
		})

		if len(exactMatches) == 0 {
			break
		}

		index++
	}

	fmt.Println("Attempting to rename lvs " + fmt.Sprint(lvs.GetID()))
	lvs.Name = lvs.Name + " (old " + fmt.Sprint(index) + ")"
	if _, err := libraryvariablesets.Update(myclient, lvs); err != nil {
		return err
	}

	return nil
}

func (s SpaceExportStep) feedExists(myclient *client.Client) (bool, feeds.IFeed, error) {
	if allFeeds, err := feeds.GetAll(myclient, myclient.GetSpaceID()); err == nil {
		filteredFeeds := lo.Filter(allFeeds, func(feed feeds.IFeed, index int) bool {
			return feed.GetName() == "Octoterra Docker Feed"
		})

		if len(filteredFeeds) != 0 {
			return true, filteredFeeds[0], nil
		}

		return false, nil, nil
	} else {
		return false, nil, err
	}
}

func (s SpaceExportStep) accountExists(myclient *client.Client, accountName string) (bool, accounts.IAccount, error) {
	if allAccounts, err := accounts.GetAll(myclient, myclient.GetSpaceID()); err == nil {
		filteredAccounts := lo.Filter(allAccounts, func(account accounts.IAccount, index int) bool {
			return account.GetName() == accountName
		})

		if len(filteredAccounts) != 0 {
			return true, filteredAccounts[0], nil
		}

		return false, nil, nil
	} else {
		return false, nil, err
	}
}
