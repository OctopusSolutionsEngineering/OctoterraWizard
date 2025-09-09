package infrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/environments"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/feeds"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/projects"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/runbooks"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/tasks"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/octoerrors"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/samber/lo"
)

const RetryCount = 5

func GetEnvironments(state state.State) ([]*environments.Environment, error) {
	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return nil, err
	}

	return environments.GetAll(myclient, myclient.GetSpaceID())
}

func WaitForTask(state state.State, taskId string, statusCallback func(message string)) error {
	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return err
	}

	// wait up to 2 hours for the task to complete
	for i := 0; i < 7200; i++ {
		mytasks, err := myclient.Tasks.Get(tasks.TasksQuery{
			Environment:             "",
			HasPendingInterruptions: false,
			HasWarningsOrErrors:     false,
			IDs:                     []string{taskId},
			IncludeSystem:           false,
			IsActive:                false,
			IsRunning:               false,
			Name:                    "",
			Node:                    "",
			PartialName:             "",
			Project:                 "",
			Runbook:                 "",
			Skip:                    0,
			Spaces:                  nil,
			States:                  nil,
			Take:                    1,
			Tenant:                  "",
		})

		if err != nil {
			return err
		}

		if len(mytasks.Items) == 0 {
			return octoerrors.TaskNotFound{TaskId: taskId}
		}

		if mytasks.Items[0].IsCompleted != nil && *mytasks.Items[0].IsCompleted {
			if mytasks.Items[0].State != "Success" {
				return octoerrors.TaskFailedError{TaskId: taskId}
			}
			statusCallback(mytasks.Items[0].State)
			return nil
		} else {
			statusCallback(mytasks.Items[0].State)
			time.Sleep(10 * time.Second)
		}
	}

	return octoerrors.TaskDidNotCompleteError{TaskId: taskId}
}

func RunRunbook(state state.State, runbookName string, projectName string, environmentName string) (string, error) {
	return RunRunbookRetry(state, runbookName, projectName, environmentName, 0, nil)
}

func RunRunbookRetry(state state.State, runbookName string, projectName string, environmentName string, retryCount int, lastError error) (string, error) {
	if retryCount > RetryCount {
		return "", errors.Join(errors.New("Failed to run runbook after "+fmt.Sprint(RetryCount)+" retries"), lastError)
	}

	if retryCount > 1 {
		time.Sleep(10 * time.Second)
	}

	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	environment, err := environments.GetAll(myclient, myclient.GetSpaceID())

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	environmentId := lo.Filter(environment, func(item *environments.Environment, index int) bool {
		return item.Name == environmentName
	})

	if len(environmentId) == 0 {
		return "", errors.Join(errors.New("Environment "+environmentName+" not found"), lastError)
	}

	project, err := projects.GetByName(myclient, myclient.GetSpaceID(), projectName)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	// The project may have been deleted
	if project == nil {
		return "", errors.New("The project " + projectName + " does not exist")
	}

	runbook, err := runbooks.GetByName(myclient, myclient.GetSpaceID(), project.GetID(), runbookName)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	if runbook == nil {
		return "", errors.New("The runbook " + runbookName + " does not exist in project " + projectName)
	}

	if runbook.PublishedRunbookSnapshotID == "" {
		return "", octoerrors.RunbookNotPublishedError{
			Runbook: runbook,
			Project: project,
		}
	}

	url := state.GetExternalServer() + runbook.GetLinks()["RunbookRunPreview"]
	url = strings.ReplaceAll(url, "{environment}", environment[0].GetID())
	url = strings.ReplaceAll(url, "{?includeDisabledSteps}", "")

	runbookRunPreviewRequest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	runbookRunPreviewResponse, err := myclient.HttpSession().DoRawRequest(runbookRunPreviewRequest)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	runbookRunPreviewRaw, err := io.ReadAll(runbookRunPreviewResponse.Body)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	if runbookRunPreviewResponse.StatusCode < 200 || runbookRunPreviewResponse.StatusCode > 299 {
		return "", octoerrors.RunbookRunFailedError{
			Runbook:  runbook,
			Project:  project,
			Response: string(runbookRunPreviewRaw),
		}
	}

	runbookRunPreview := map[string]any{}
	err = json.Unmarshal(runbookRunPreviewRaw, &runbookRunPreview)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	runbookFormNames := lo.Map(runbookRunPreview["Form"].(map[string]any)["Elements"].([]any), func(value any, index int) any {
		return value.(map[string]any)["Name"]
	})

	runbookFormValues := map[string]string{}

	for _, name := range runbookFormNames {
		// OctoterraWiz.Destination.ProjectName is a special variable that is used to define the name of the destination project.
		if name.(string) != "OctoterraWiz.Destination.ProjectName" {
			runbookFormValues[name.(string)] = "dummy"
		}
	}

	runbookBody := map[string]any{
		"RunbookId":                runbook.GetID(),
		"RunbookSnapShotId":        runbook.PublishedRunbookSnapshotID,
		"FrozenRunbookProcessId":   nil,
		"EnvironmentId":            environmentId[0].ID,
		"TenantId":                 nil,
		"SkipActions":              []string{},
		"QueueTime":                nil,
		"QueueTimeExpiry":          nil,
		"FormValues":               runbookFormValues,
		"ForcePackageDownload":     false,
		"ForcePackageRedeployment": true,
		"UseGuidedFailure":         false,
		"SpecificMachineIds":       []string{},
		"ExcludedMachineIds":       []string{},
	}

	runbookBodyJson, err := json.Marshal(runbookBody)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	url = state.GetExternalServer() + "/api/" + state.Space + "/runbookRuns"
	runbookRunRequest, err := http.NewRequest("POST", url, bytes.NewReader(runbookBodyJson))

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	runbookRunResponse, err := myclient.HttpSession().DoRawRequest(runbookRunRequest)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	runbookRunRaw, err := io.ReadAll(runbookRunResponse.Body)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	if runbookRunResponse.StatusCode < 200 || runbookRunResponse.StatusCode > 299 {
		return "", octoerrors.RunbookRunFailedError{
			Runbook:  runbook,
			Project:  project,
			Response: string(runbookRunRaw),
		}
	}

	runbookRun := map[string]any{}
	err = json.Unmarshal(runbookRunRaw, &runbookRun)

	if err != nil {
		return RunRunbookRetry(state, runbookName, projectName, environmentName, retryCount+1, err)
	}

	if _, ok := runbookRun["TaskId"]; !ok {
		return "", octoerrors.RunbookRunFailedError{Runbook: runbook, Project: project, Response: string(runbookRunRaw)}
	}

	return runbookRun["TaskId"].(string), nil

}

func PublishRunbook(state state.State, runbookName string, projectName string) error {
	return PublishRunbookRetry(state, runbookName, projectName, 0, nil)
}

func PublishRunbookRetry(state state.State, runbookName string, projectName string, retryCount int, lastError error) error {
	if retryCount > RetryCount {
		return errors.Join(errors.New("Failed to publish runbook after "+fmt.Sprint(RetryCount)+" retries"), lastError)
	}

	if retryCount > 1 {
		time.Sleep(10 * time.Second)
	}

	myclient, err := octoclient.CreateClient(state)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	project, err := projects.GetByName(myclient, myclient.GetSpaceID(), projectName)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	// The project may have been deleted
	if project == nil {
		return errors.New("The project " + projectName + " does not exist")
	}

	runbook, err := runbooks.GetByName(myclient, myclient.GetSpaceID(), project.GetID(), runbookName)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	// The project may have been deleted
	if runbook == nil {
		return errors.New("The runbook " + runbookName + " does not exist in project " + projectName)
	}

	url := state.GetExternalServer() + runbook.GetLinks()["RunbookSnapshotTemplate"]
	runbookSnapshotTemplateRequest, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	runbookSnapshotTemplateResponse, err := myclient.HttpSession().DoRawRequest(runbookSnapshotTemplateRequest)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	runbookSnapshotTemplateRaw, err := io.ReadAll(runbookSnapshotTemplateResponse.Body)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	if runbookSnapshotTemplateResponse.StatusCode < 200 || runbookSnapshotTemplateResponse.StatusCode > 299 {
		return octoerrors.RunbookPublishFailedError{
			Runbook:  runbook,
			Project:  project,
			Response: string(runbookSnapshotTemplateRaw),
		}
	}

	runbookSnapshotTemplate := map[string]any{}
	err = json.Unmarshal(runbookSnapshotTemplateRaw, &runbookSnapshotTemplate)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	snapshot := map[string]any{
		"ProjectId": project.GetID(),
		"RunbookId": runbook.GetID(),
		"Name":      runbookSnapshotTemplate["NextNameIncrement"],
	}

	var packageErrors error = nil
	snapshot["SelectedPackages"] = lo.Map(runbookSnapshotTemplate["Packages"].([]any), func(pkg any, index int) any {
		snapshotPackage := pkg.(map[string]any)
		versions, err := feeds.SearchPackageVersions(myclient, myclient.GetSpaceID(), snapshotPackage["FeedId"].(string), snapshotPackage["PackageId"].(string), "", 1)

		if err != nil {
			packageErrors = errors.Join(packageErrors, err)
			return nil
		}
		// Check for the possibility of no package versions being returned
		if len(versions.Items) == 0 {
			packageErrors = errors.Join(packageErrors, fmt.Errorf(
				"No versions found for package '%s' in feed '%s' in runbook snapshot",
				snapshotPackage["PackageId"],
				snapshotPackage["FeedId"],
			))
			return nil
		}

		return map[string]any{
			"ActionName":           snapshotPackage["ActionName"],
			"Version":              versions.Items[0].Version,
			"PackageReferenceName": snapshotPackage["PackageReferenceName"],
		}
	})

	if packageErrors != nil {
		return packageErrors
	}

	snapshotJson, err := json.Marshal(snapshot)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	url = state.GetExternalServer() + "/api/" + state.Space + "/runbookSnapshots?publish=true"
	runbookSnapshotRequest, err := http.NewRequest("POST", url+"?publish=true", bytes.NewBuffer(snapshotJson))

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	runbookSnapshotResponse, err := myclient.HttpSession().DoRawRequest(runbookSnapshotRequest)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	runbookSnapshotResponseRaw, err := io.ReadAll(runbookSnapshotResponse.Body)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	if runbookSnapshotResponse.StatusCode < 200 || runbookSnapshotResponse.StatusCode > 299 {
		return octoerrors.RunbookPublishFailedError{
			Runbook:  runbook,
			Project:  project,
			Response: string(runbookSnapshotResponseRaw),
		}
	}

	runbookSnapshot := map[string]any{}
	err = json.Unmarshal(runbookSnapshotResponseRaw, &runbookSnapshot)

	if err != nil {
		return PublishRunbookRetry(state, runbookName, projectName, retryCount+1, err)
	}

	fmt.Println(runbookSnapshot)

	return nil
}

// GetProjects gets all projects, excluding the "Octoterra Space Management" project and any
// that are disabled.
func GetProjects(myclient *client.Client) ([]*projects.Project, error) {
	if allprojects, err := myclient.Projects.GetAll(); err != nil {
		return nil, errors.Join(errors.New("failed to get all projects"), err)
	} else {
		filteredProjects := lo.Filter(allprojects, func(item *projects.Project, index int) bool {
			return !item.IsDisabled && item.Name != "Octoterra Space Management"
		})
		return filteredProjects, nil
	}
}
