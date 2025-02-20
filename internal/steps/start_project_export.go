package steps

import (
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/deployments"
	projects2 "github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/mcasperson/OctoterraWizard/internal/data"
	"github.com/mcasperson/OctoterraWizard/internal/infrastructure"
	"github.com/mcasperson/OctoterraWizard/internal/logutil"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/octoerrors"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"github.com/samber/lo"
	"net/url"
)

type StartProjectExportStep struct {
	BaseStep
	Wizard         wizard.Wizard
	exportProjects *widget.Button
	logs           *widget.Entry
	exportDone     bool
}

func (s StartProjectExportStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, previous, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(StartSpaceExportStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		moveNext := func(proceed bool) {
			if !proceed {
				return
			}

			s.Wizard.ShowWizardStep(FinishStep{
				Wizard:   s.Wizard,
				BaseStep: BaseStep{State: s.State}})
		}
		if !s.exportDone {
			dialog.NewConfirm(
				"Do you want to skip this step?",
				"You can run the runbooks manually from the Octopus UI.", moveNext, s.Wizard.Window).Show()
		} else {
			moveNext(true)
		}
	})
	linkUrl, _ := url.Parse(s.State.Server + "/app#/" + s.State.Space + "/tasks")
	link := widget.NewHyperlink("View the task list", linkUrl)
	link.Hide()
	s.logs = widget.NewEntry()
	s.logs.SetMinRowsVisible(20)
	s.logs.Disable()
	s.logs.Hide()
	s.logs.MultiLine = true
	s.exportDone = false

	heading := widget.NewLabel("Migrate Projects")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		The projects in the source space are now ready to begin exporting to the destination space.
		This involves serializing the project level resources (project, runbooks, variables, triggers etc) to a Terraform module and then applying the module to the destination space.
		First, this wizard publishes and runs the "__ 1. Serialize Project" runbook in each project to create the Terraform module.
		Then this wizard publishes and runs the "__ 2. Deploy Project" runbook in each project to apply the Terraform module to the destination space.
		Click the "Export Projects" button to execute these runbooks.
	`))
	result := widget.NewLabel("")
	infinite := widget.NewProgressBarInfinite()
	infinite.Hide()
	infinite.Start()
	s.exportProjects = widget.NewButton("Export Projects", func() {
		s.exportProjects.Disable()
		next.Disable()
		previous.Disable()
		infinite.Show()
		link.Hide()
		s.exportDone = true
		defer s.exportProjects.Enable()
		defer previous.Enable()
		defer next.Enable()
		defer infinite.Hide()

		result.SetText("🔵 Running the runbooks.")

		if err := s.Execute(func(message string) {
			result.SetText(message)
		}); err != nil {
			if err := logutil.WriteTextToFile("start_project_export_error.txt", err.Error()); err != nil {
				fmt.Println("Failed to write error to file")
			}

			result.SetText(fmt.Sprintf("🔴 Failed to publish and run the runbooks. The failed tasks are shown below. You can review the task details in the Octopus console to find more information."))
			s.logs.SetText(err.Error())
			s.logs.Show()
			link.Show()
		} else {
			result.SetText("🟢 Runbooks ran successfully.")
			next.Enable()
			s.logs.Hide()
		}
	})
	middle := container.New(layout.NewVBoxLayout(), heading, label1, s.exportProjects, infinite, result, link, s.logs)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s StartProjectExportStep) Execute(statusCallback func(message string)) error {
	myclient, err := octoclient.CreateClient(s.State)

	if err != nil {
		return errors.Join(errors.New("failed to create client"), err)
	}

	projects, err := projects2.GetAll(myclient, myclient.GetSpaceID())

	if err != nil {
		return errors.Join(errors.New("failed to get all projects"), err)
	}

	filteredProjects := lo.Filter(projects, func(project *projects2.Project, index int) bool {
		return project.Name != "Octoterra Space Management"
	})

	// We start by exporting projects that do not have "Deploy a release" steps
	var filterErrors error = nil
	regularProjects := lo.Filter(filteredProjects, func(project *projects2.Project, index int) bool {

		var process *deployments.DeploymentProcess = nil

		if project.IsVersionControlled {
			process, err = deployments.GetDeploymentProcessByGitRef(myclient, myclient.GetSpaceID(), project, "refs/heads/main")

			if err != nil {
				filterErrors = errors.Join(filterErrors, errors.Join(errors.New("failed to get deployment process by gitref \"refs/heads/main\""), err))
				return false
			}
		} else {
			process, err = deployments.GetDeploymentProcessByID(myclient, myclient.GetSpaceID(), project.DeploymentProcessID)

			if err != nil {
				filterErrors = errors.Join(filterErrors, errors.Join(errors.New("failed to get deployment process by ID "+project.DeploymentProcessID), err))
				return false
			}
		}

		return !lo.ContainsBy(process.Steps, func(step *deployments.DeploymentStep) bool {
			return lo.ContainsBy(step.Actions, func(action *deployments.DeploymentAction) bool {
				return action.ActionType == "Octopus.DeployRelease"
			})
		})
	})

	if filterErrors != nil {
		return filterErrors
	}

	runAndTaskError := s.serializeProjects(regularProjects, statusCallback)
	runAndTaskError = errors.Join(runAndTaskError, s.deployProjects(regularProjects, statusCallback))

	// Now we export projects that have "Deploy a release" steps. This ensures any child projects are available to
	// be queried via a data source in the Terraform module.
	deployReleaseProjects := lo.Filter(filteredProjects, func(project *projects2.Project, index int) bool {
		return !lo.ContainsBy(regularProjects, func(regularProject *projects2.Project) bool {
			return project.ID == regularProject.ID
		})
	})

	runAndTaskError = errors.Join(runAndTaskError, s.serializeProjects(deployReleaseProjects, statusCallback))
	runAndTaskError = errors.Join(runAndTaskError, s.deployProjects(deployReleaseProjects, statusCallback))

	return runAndTaskError
}

func (s StartProjectExportStep) serializeProjects(filteredProjects []*projects2.Project, statusCallback func(message string)) error {
	var runAndTaskError error = nil

	for _, project := range filteredProjects {
		if err := infrastructure.PublishRunbook(s.State, "__ 1. Serialize Project", project.Name); err != nil {
			return errors.Join(errors.New("failed to publish runbook \"__ 1. Serialize Project\" for project "+project.Name), err)
		}

		statusCallback("🔵 Published __ 1. Serialize Project runbook in project " + project.Name)
	}

	tasks := []data.NameValuePair{}

	for _, project := range filteredProjects {
		if taskId, err := infrastructure.RunRunbook(s.State, "__ 1. Serialize Project", project.Name); err != nil {

			var failedRunbookRun octoerrors.RunbookRunFailedError
			if errors.As(err, &failedRunbookRun) {
				runAndTaskError = errors.Join(runAndTaskError, errors.Join(errors.New("failed to run runbook \"__ 1. Serialize Project\" in project "+project.Name), failedRunbookRun))
			} else {
				return errors.Join(errors.New("failed to run runbook \"__ 1. Serialize Project\" for project "+project.Name), err)
			}
		} else {
			tasks = append(tasks, data.NameValuePair{Name: project.Name, Value: taskId})
		}
	}

	serializeIndex := 0
	statusCallback("🔵 Started running the __ 1. Serialize Project runbooks (" + fmt.Sprint(serializeIndex) + "/" + fmt.Sprint(len(tasks)) + ")")
	for _, task := range tasks {
		if err := infrastructure.WaitForTask(s.State, task.Value, func(message string) {
			statusCallback("🔵 __ 1. Serialize Project for project " + task.Name + " is " + message + " (" + fmt.Sprint(serializeIndex) + "/" + fmt.Sprint(len(tasks)) + ")")
		}); err != nil {
			runAndTaskError = errors.Join(runAndTaskError, errors.Join(errors.New("failed to get task state for task "+task.Name), err))
		}
		serializeIndex++
	}

	return runAndTaskError
}

func (s StartProjectExportStep) deployProjects(filteredProjects []*projects2.Project, statusCallback func(message string)) error {
	var runAndTaskError error = nil

	for _, project := range filteredProjects {
		if err := infrastructure.PublishRunbook(s.State, "__ 2. Deploy Project", project.Name); err != nil {
			return errors.Join(errors.New("failed to publish runbook \"__ 2. Deploy Project\" for project "+project.Name), err)
		}
		statusCallback("🔵 Published __ 2. Deploy Space runbook in project " + project.Name)
	}

	applyTasks := []data.NameValuePair{}
	for _, project := range filteredProjects {
		if taskId, err := infrastructure.RunRunbook(s.State, "__ 2. Deploy Project", project.Name); err != nil {
			var failedRunbookRun octoerrors.RunbookRunFailedError
			if errors.As(err, &failedRunbookRun) {
				runAndTaskError = errors.Join(runAndTaskError, errors.Join(errors.New("failed to run runbook \"__ 2. Deploy Project\" in project "+project.Name), failedRunbookRun))
			} else {
				return errors.Join(errors.New("Failed to run runbook \"__ 2. Deploy Project\" for project "+project.Name), err)
			}
		} else {
			applyTasks = append(applyTasks, data.NameValuePair{Name: project.Name, Value: taskId})
		}
	}

	applyIndex := 0
	statusCallback("🔵 Started running the __ 2. Deploy Project runbooks (" + fmt.Sprint(applyIndex) + "/" + fmt.Sprint(len(applyTasks)) + ")")
	for _, task := range applyTasks {
		if err := infrastructure.WaitForTask(s.State, task.Value, func(message string) {
			statusCallback("🔵 __ 2. Deploy Project for project " + task.Name + " is " + message + " (" + fmt.Sprint(applyIndex) + "/" + fmt.Sprint(len(applyTasks)) + ")")
		}); err != nil {
			runAndTaskError = errors.Join(runAndTaskError, errors.Join(errors.New("failed to get task state for task "+task.Name), err))
		}
		applyIndex++
	}

	return runAndTaskError
}
