package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/validators"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"net/url"
	"strings"
)

type OctopusDestinationDetails struct {
	BaseStep
	Wizard  wizard.Wizard
	server  *widget.Entry
	apiKey  *widget.Entry
	spaceId *widget.Entry
	result  *widget.Label
}

func (s OctopusDestinationDetails) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, _, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(OctopusDetails{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.getState()}})
	}, func() {
		s.result.SetText("🔵 Validating Octopus credentials.")
		s.apiKey.Disable()
		s.server.Disable()
		s.spaceId.Disable()
		defer s.apiKey.Enable()
		defer s.server.Enable()
		defer s.spaceId.Enable()

		validationFailed := false
		if !validators.ValidateDestinationCreds(s.getState()) {
			s.result.SetText("🔴 Unable to connect to the Octopus server. Please check the URL, API key, and Space ID.")
			validationFailed = true
		}

		nexCallback := func(proceed bool) {
			if proceed {
				s.Wizard.ShowWizardStep(ToolsSelectionStep{
					Wizard:   s.Wizard,
					BaseStep: BaseStep{State: s.getState()}})
			}
		}

		if validationFailed {
			dialog.NewConfirm("Octopus Validation failed", "Validation of the Octopus details failed. Do you wish to continue anyway?", nexCallback, s.Wizard.Window).Show()
		} else {
			nexCallback(true)
		}
	})

	s.result = widget.NewLabel("")

	validation := func(input string) {
		next.Disable()

		if s.server != nil && s.server.Text != "" && s.apiKey != nil && s.apiKey.Text != "" && s.spaceId != nil && s.spaceId.Text != "" {
			return
		}

		next.Enable()
	}

	validation("")

	heading := widget.NewLabel("Octopus Destination Server")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	introText := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Enter the URL, API key, and Space ID of the Octopus instance you want to export to (i.e. the destination server).
		Note that all the resources in the destination space must be managed by Terraform.
		Typically this means the destination space must be blank and all resources are created by running this wizard.
		Terraform will not replace or update existing resources by default.`))
	linkUrl, _ := url.Parse("https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key")
	link := widget.NewHyperlink("Learn how to create an API key.", linkUrl)

	serverLabel := widget.NewLabel("Destination Server URL")
	s.server = widget.NewEntry()
	s.server.SetPlaceHolder("https://octopus.example.com")
	s.server.SetText(s.State.DestinationServer)

	apiKeyLabel := widget.NewLabel("Destination API Key")
	s.apiKey = widget.NewPasswordEntry()
	s.apiKey.SetPlaceHolder("API-xxxxxxxxxxxxxxxxxxxxxxxxxx")
	s.apiKey.SetText(s.State.DestinationApiKey)

	spaceIdLabel := widget.NewLabel("Destination Space ID")
	s.spaceId = widget.NewEntry()
	s.spaceId.SetPlaceHolder("Spaces-#")
	s.spaceId.SetText(s.State.DestinationSpace)

	s.server.OnChanged = validation
	s.apiKey.OnChanged = validation
	s.spaceId.OnChanged = validation

	formLayout := container.New(layout.NewFormLayout(), serverLabel, s.server, apiKeyLabel, s.apiKey, spaceIdLabel, s.spaceId)

	middle := container.New(layout.NewVBoxLayout(), heading, introText, link, formLayout, s.result)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s OctopusDestinationDetails) getState() state.State {
	return state.State{
		BackendType:               s.State.BackendType,
		Server:                    s.State.Server,
		ServerExternal:            "",
		ApiKey:                    s.State.ApiKey,
		Space:                     s.State.Space,
		DestinationServer:         strings.TrimSpace(s.server.Text),
		DestinationServerExternal: "",
		DestinationApiKey:         strings.TrimSpace(s.apiKey.Text),
		DestinationSpace:          strings.TrimSpace(s.spaceId.Text),
		AwsAccessKey:              s.State.AwsAccessKey,
		AwsSecretKey:              s.State.AwsSecretKey,
		AwsS3Bucket:               s.State.AwsS3Bucket,
		AwsS3BucketRegion:         s.State.AwsS3BucketRegion,
		PromptForDelete:           s.State.PromptForDelete,
		UseContainerImages:        s.State.UseContainerImages,
		AzureResourceGroupName:    s.State.AzureResourceGroupName,
		AzureStorageAccountName:   s.State.AzureStorageAccountName,
		AzureContainerName:        s.State.AzureContainerName,
		AzureSubscriptionId:       s.State.AzureSubscriptionId,
		AzureTenantId:             s.State.AzureTenantId,
		AzureApplicationId:        s.State.AzureApplicationId,
		AzurePassword:             s.State.AzurePassword,
	}
}
