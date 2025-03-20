package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/client"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/spaces"
	"github.com/OctopusDeploy/go-octopusdeploy/v2/pkg/variables"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/samber/lo"
	"io"
	"net/http"
)

type CommunityStepTemplates struct {
	Items []CommunityStepTemplate `json:"Items"`
}

type CommunityStepTemplate struct {
	Id      string `json:"Id"`
	Website string `json:"Website"`
}

type StepTemplates struct {
	Items []StepTemplate `json:"Items"`
}

type StepTemplate struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}

func GetSpaceName(myclient *client.Client, state state.State) (string, error) {
	space, err := spaces.GetByID(myclient, state.Space)

	if err != nil {
		return "", err
	}

	return space.Name, nil
}

func GetStepTemplateId(myclient *client.Client, state state.State, name string) (string, error, string) {
	var body io.Reader
	req, err := http.NewRequest("GET", state.GetExternalServer()+"/api/"+state.Space+"/actiontemplates?take=10000", body)

	if err != nil {
		return "", err, "🔴 Failed to create the step templates request"
	}

	response, err := myclient.HttpSession().DoRawRequest(req)

	if err != nil {
		return "", err, "🔴 Failed to get the step templates"
	}

	responseBody, err := io.ReadAll(response.Body)

	if err != nil {
		return "", err, "🔴 Failed to read the step templates query body"
	}

	stepTemplates := StepTemplates{}
	if err := json.Unmarshal(responseBody, &stepTemplates); err != nil {
		return "", err, "🔴 Failed to unmarshal the step templates response"
	}

	if stepTemplates.Items == nil {
		stepTemplates.Items = []StepTemplate{}
	}

	filteredStepTemplates := lo.Filter(stepTemplates.Items, func(stepTemplate StepTemplate, index int) bool {
		return stepTemplate.Name == name
	})

	if len(filteredStepTemplates) == 0 {
		return "", errors.New("could not find the step template"), "🔴 Failed to find the step template called " + name
	}

	return filteredStepTemplates[0].Id, nil, ""
}

func LibraryVariableSetExists(myclient *client.Client, name string) (bool, *variables.LibraryVariableSet, error) {
	if resource, err := myclient.LibraryVariableSets.GetByPartialName("Octoterra"); err == nil {
		exatchMatch := lo.Filter(resource, func(item *variables.LibraryVariableSet, index int) bool {
			return item.Name == name
		})

		if len(exatchMatch) == 0 {
			return false, nil, nil
		}

		return true, exatchMatch[0], nil
	} else {
		return false, nil, err
	}
}

func InstallStepTemplate(myclient *client.Client, state state.State, website string) (error, string) {
	var body io.Reader
	req, err := http.NewRequest("GET", state.GetExternalServer()+"/api/communityactiontemplates?take=10000", body)

	if err != nil {
		return err, "🔴 Failed to get the community step templates"
	}

	response, err := myclient.HttpSession().DoRawRequest(req)

	if err != nil {
		return err, "🔴 Failed to get the step templates"
	}

	responseBody, err := io.ReadAll(response.Body)

	//fmt.Print(string(responseBody))

	if err != nil {
		return err, "🔴 Failed to read the step templates query body"
	}

	stepTemplates := CommunityStepTemplates{}
	if err := json.Unmarshal(responseBody, &stepTemplates); err != nil {
		return err, "🔴 Failed to unmarshal the step templates response"
	}

	if stepTemplates.Items == nil {
		stepTemplates.Items = []CommunityStepTemplate{}
	}

	serializeSpaceTemplate := lo.Filter(stepTemplates.Items, func(stepTemplate CommunityStepTemplate, index int) bool {
		return stepTemplate.Website == website
	})

	if len(serializeSpaceTemplate) == 0 {
		return errors.New("did not find step template"), "🔴 Failed to find the step template"
	}

	var installBody io.Reader
	installReq, err := http.NewRequest("POST", state.GetExternalServer()+"/api/communityactiontemplates/"+serializeSpaceTemplate[0].Id+"/installation/"+state.Space, installBody)

	if err != nil {
		return err, "🔴 Failed to create the request to install the community step templates"
	}

	installResp, err := myclient.HttpSession().DoRawRequest(installReq)

	if err != nil {
		return err, "🔴 Failed to install the community step templates"
	}

	installResponseBody, err := io.ReadAll(installResp.Body)

	fmt.Print(string(installResponseBody))

	return nil, ""
}
