package steps

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/runbooks"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/query"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"github.com/samber/lo"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed modules/project_management/terraform.tf
var runbookModule string

type ProjectExportStep struct {
	BaseStep
	Wizard        wizard.Wizard
	createProject *widget.Button
	infinite      *widget.ProgressBarInfinite
	result        *widget.Label
	logs          *widget.Entry
	next          *widget.Button
	previous      *widget.Button
}

func (s ProjectExportStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, thisPrevious, thisNext := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(SpaceExportStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		s.Wizard.ShowWizardStep(StartSpaceExportStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	})
	s.next = thisNext
	s.previous = thisPrevious
	s.next.Disable()

	intro := widget.NewLabel(strutil.TrimMultilineWhitespace(`Each project gets two runbooks, one to serialize it to a Terraform module, and the second to deploy it.`))
	s.infinite = widget.NewProgressBarInfinite()
	s.infinite.Start()
	s.infinite.Hide()
	s.result = widget.NewLabel("")
	s.logs = widget.NewEntry()
	s.logs.Disable()
	s.logs.MultiLine = true
	s.logs.SetMinRowsVisible(20)
	s.createProject = widget.NewButton("Add Runbooks", func() { s.createNewProject(parent) })
	middle := container.New(layout.NewVBoxLayout(), intro, s.createProject, s.infinite, s.result, s.logs)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s ProjectExportStep) createNewProject(parent fyne.Window) {
	s.result.SetText("")
	s.logs.SetText("")
	s.next.Disable()
	s.previous.Disable()
	s.infinite.Show()
	s.createProject.Disable()
	s.result.SetText("🔵 Creating runbooks. This can take a little while.")

	s.Execute(func(title string, message string, callback func(bool)) {
		dialog.NewConfirm(title, message, callback, parent).Show()
	}, func(message string, err error) {
		s.result.SetText(message)
		s.logs.SetText(err.Error())
		s.previous.Enable()
		s.next.Disable()
		s.infinite.Hide()
		s.createProject.Enable()
	}, func(message string) {
		s.result.SetText(message)
		s.logs.SetText("")
		s.next.Enable()
		s.previous.Enable()
		s.infinite.Hide()
		s.createProject.Enable()
	})
}

func (s ProjectExportStep) Execute(prompt func(string, string, func(bool)), handleError func(string, error), handleSuccess func(string)) {
	myclient, err := octoclient.CreateClient(s.State)

	if err != nil {
		handleError("🔴 Failed to create the client", err)
		return
	}

	allProjects, err := s.getProjects(myclient)

	if err != nil {
		handleError("🔴 Failed to get all the projects", err)
		return
	}

	allProjects = lo.Filter(allProjects, func(project *projects.Project, index int) bool {
		return project.Name != spaceManagementProject
	})

	lvsExists, lvs, err := query.LibraryVariableSetExists(myclient)

	if err != nil {
		handleError("🔴 Failed to get the library variable set Octoterra", err)
		return
	}

	if !lvsExists {
		handleError("🔴 The library variable set Octoterra could not be found", errors.New("resource not found"))
		return
	}

	// First look deletes any existing projects
	for _, project := range allProjects {
		if project.Name == spaceManagementProject {
			continue
		}

		runbookExists, runbook, err := s.runbookExists(myclient, project.ID, "__ 1. Serialize Project")

		if err != nil {
			handleError("🔴 Failed to find runbook", err)
			return
		}

		if runbookExists {
			deleteRunbook1Func := func(b bool) {
				if b {
					if err := s.deleteRunbook(myclient, runbook); err != nil {
						s.result.SetText("🔴 Failed to delete the resource")
						s.logs.SetText(err.Error())
					} else if s.State.PromptForDelete {
						s.Execute(prompt, handleError, handleSuccess)
					}
				}
			}

			if s.State.PromptForDelete {
				prompt("Project Group Exists", "The runbook \""+runbook.Name+"\" already exists in project "+project.Name+". Do you want to delete it? It is usually safe to delete this resource.", deleteRunbook1Func)
				return
			} else {
				deleteRunbook1Func(true)
			}
		}

		runbook2Exists, runbook2, err := s.runbookExists(myclient, project.ID, "__ 2. Deploy Project")

		if err != nil {
			handleError("🔴 Failed to find runbook", err)
			return
		}

		if runbook2Exists {
			deleteRunbook2Func := func(b bool) {
				if b {
					if err := s.deleteRunbook(myclient, runbook2); err != nil {
						s.result.SetText("🔴 Failed to delete the resource")
						s.logs.SetText(err.Error())
					} else if s.State.PromptForDelete {
						s.Execute(prompt, handleError, handleSuccess)
					}
				}
			}

			if s.State.PromptForDelete {
				prompt("Project Group Exists", "The runbook \""+runbook2.Name+"\" already exists in project "+project.Name+". Do you want to delete it? It is usually safe to delete this resource.", deleteRunbook2Func)
				return
			} else {
				deleteRunbook2Func(true)
			}
		}
	}

	// Find the step template ID
	serializeProjectTemplate, err, message := query.GetStepTemplateId(myclient, s.State, "Octopus - Serialize Project to Terraform")

	if err != nil {
		handleError(message, err)
		return
	}

	deploySpaceTemplateS3, err, message := query.GetStepTemplateId(myclient, s.State, "Octopus - Populate Octoterra Space (S3 Backend)")

	if err != nil {
		handleError(message, err)
		return
	}

	for index, project := range allProjects {
		// Save and apply the module
		dir, err := ioutil.TempDir("", "octoterra")
		if err != nil {
			handleError("🔴 An error occurred while creating a temporary directory", err)
			return
		}

		filePath := filepath.Join(dir, "terraform.tf")

		if err := os.WriteFile(filePath, []byte(runbookModule), 0644); err != nil {
			handleError("🔴 An error occurred while writing the Terraform file", err)
			return
		}

		initCmd := exec.Command("terraform", "init", "-no-color")
		initCmd.Dir = dir

		var initStdout, initStderr bytes.Buffer
		initCmd.Stdout = &initStdout
		initCmd.Stderr = &initStderr

		if err := initCmd.Run(); err != nil {
			handleError("🔴 Terraform init failed.", errors.New(initStdout.String()+initCmd.String()))
			return
		}

		applyCmd := exec.Command("terraform",
			"apply",
			"-auto-approve",
			"-no-color",
			"-var=octopus_serialize_actiontemplateid="+serializeProjectTemplate,
			"-var=octopus_deploys3_actiontemplateid="+deploySpaceTemplateS3,
			"-var=octopus_server="+s.State.Server,
			"-var=octopus_apikey="+s.State.ApiKey,
			"-var=octopus_space_id="+s.State.Space,
			"-var=octopus_project_id="+project.ID,
			"-var=terraform_state_bucket="+s.State.AwsS3Bucket,
			"-var=terraform_state_bucket_region="+s.State.AwsS3BucketRegion,
			"-var=octopus_destination_server="+s.State.DestinationServer,
			"-var=octopus_destination_apikey="+s.State.DestinationApiKey,
			"-var=octopus_destination_space_id="+s.State.DestinationSpace,
			"-var=octopus_project_name="+project.Name)
		applyCmd.Dir = dir

		var stdout, stderr bytes.Buffer
		applyCmd.Stdout = &stdout
		applyCmd.Stderr = &stderr

		if err := applyCmd.Run(); err != nil {
			handleError("🔴 Terraform apply failed", errors.New(stdout.String()+stderr.String()))
			return
		} else {
			handleSuccess("🔵 Terraform apply succeeded (" + fmt.Sprint(index) + " / " + fmt.Sprint(len(allProjects)) + ")")
		}

		// link the library variable set
		project, err := myclient.Projects.GetByID(project.ID)

		if err != nil {
			handleError("🔴 Failed to get the project", err)
			return
		}

		project.IncludedLibraryVariableSets = append(project.IncludedLibraryVariableSets, lvs.ID)

		_, err = projects.Update(myclient, project)

		if err != nil {
			handleError("🔴 Failed to update the project", err)
			return
		}

	}

	handleSuccess("🟢 Added runbooks to all projects")
}

func (s ProjectExportStep) getProjects(myclient *client.Client) ([]*projects.Project, error) {
	if allprojects, err := myclient.Projects.GetAll(); err != nil {
		return nil, err
	} else {
		return allprojects, nil
	}
}

func (s ProjectExportStep) deleteRunbook(myclient *client.Client, runbook *runbooks.Runbook) error {
	if err := myclient.Runbooks.DeleteByID(runbook.ID); err != nil {
		return err
	}

	return nil
}

func (s ProjectExportStep) runbookExists(myclient *client.Client, projectId string, runbookName string) (bool, *runbooks.Runbook, error) {
	if runbook, err := runbooks.GetByName(myclient, myclient.GetSpaceID(), projectId, runbookName); err == nil {
		if runbook == nil {
			return false, nil, nil
		}
		return true, runbook, nil
	} else {
		return false, nil, err
	}
}
