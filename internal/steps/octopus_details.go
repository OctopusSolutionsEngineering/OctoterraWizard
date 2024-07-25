package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/mcasperson/OctoterraWizard/internal/validators"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"net/url"
	"strings"
)

type OctopusDetails struct {
	BaseStep
	Wizard  wizard.Wizard
	server  *widget.Entry
	apiKey  *widget.Entry
	spaceId *widget.Entry
	result  *widget.Label
}

func (s OctopusDetails) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, _, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(TestTerraformStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.getState()}})
	}, func() {
		s.result.SetText("🔵 Validating Octopus credentials.")
		s.apiKey.Disable()
		s.server.Disable()
		s.spaceId.Disabled()
		defer s.apiKey.Enable()
		defer s.server.Enable()
		defer s.spaceId.Enable()

		if !validators.ValidateSourceCreds(s.getState()) {
			s.result.SetText("🔴 Unable to connect to the Octopus server. Please check the URL, API key, and Space ID.")
			return
		}

		s.Wizard.ShowWizardStep(OctopusDestinationDetails{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.getState()}})
	})

	s.result = widget.NewLabel("")

	validation := func(input string) {
		next.Disabled()

		if s.server != nil && s.server.Text != "" && s.apiKey != nil && s.apiKey.Text != "" && s.spaceId != nil && s.spaceId.Text != "" {
			return
		}

		next.Enable()
	}

	validation("")

	introText := widget.NewLabel("Enter the URL, API key, and Space ID of the Octopus instance you want to export from (i.e. the source server).")
	linkUrl, _ := url.Parse("https://octopus.com/docs/octopus-rest-api/how-to-create-an-api-key")
	link := widget.NewHyperlink("Learn how to create an API key.", linkUrl)

	serverLabel := widget.NewLabel("Source Server URL")
	s.server = widget.NewEntry()
	s.server.SetPlaceHolder("https://octopus.example.com")
	s.server.SetText(s.State.Server)

	apiKeyLabel := widget.NewLabel("Source API Key")
	s.apiKey = widget.NewPasswordEntry()
	s.apiKey.SetPlaceHolder("API-xxxxxxxxxxxxxxxxxxxxxxxxxx")
	s.apiKey.SetText(s.State.ApiKey)

	spaceIdLabel := widget.NewLabel("Source Space ID")
	s.spaceId = widget.NewEntry()
	s.spaceId.SetPlaceHolder("Spaces-#")
	s.spaceId.SetText(s.State.Space)

	s.server.OnChanged = validation
	s.apiKey.OnChanged = validation
	s.spaceId.OnChanged = validation

	formLayout := container.New(layout.NewFormLayout(), serverLabel, s.server, apiKeyLabel, s.apiKey, spaceIdLabel, s.spaceId)

	middle := container.New(layout.NewVBoxLayout(), introText, link, formLayout, s.result)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s OctopusDetails) getState() state.State {
	return state.State{
		BackendType:             s.State.BackendType,
		Server:                  strings.TrimSpace(s.server.Text),
		ApiKey:                  strings.TrimSpace(s.apiKey.Text),
		Space:                   strings.TrimSpace(s.spaceId.Text),
		DestinationServer:       s.State.DestinationServer,
		DestinationApiKey:       s.State.DestinationApiKey,
		DestinationSpace:        s.State.DestinationSpace,
		AwsAccessKey:            s.State.AwsAccessKey,
		AwsSecretKey:            s.State.AwsSecretKey,
		AwsS3Bucket:             s.State.AwsS3Bucket,
		AwsS3BucketRegion:       s.State.AwsS3BucketRegion,
		PromptForDelete:         s.State.PromptForDelete,
		AzureResourceGroupName:  s.State.AzureResourceGroupName,
		AzureStorageAccountName: s.State.AzureStorageAccountName,
		AzureContainerName:      s.State.AzureContainerName,
		AzureSubscriptionId:     s.State.AzureSubscriptionId,
		AzureTenantId:           s.State.AzureTenantId,
		AzureApplicationId:      s.State.AzureApplicationId,
		AzurePassword:           s.State.AzurePassword,
	}
}
