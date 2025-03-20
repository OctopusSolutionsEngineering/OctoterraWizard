package steps

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/logutil"
	"github.com/mcasperson/OctoterraWizard/internal/sensitivevariables"
	"github.com/mcasperson/OctoterraWizard/internal/state"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/validators"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"net/url"
	"strings"
)

// ExtractSecrets provides a step in the wizard to extract secrets from the Octopus database
type ExtractSecrets struct {
	BaseStep
	Wizard    wizard.Wizard
	dbServer  *widget.Entry
	port      *widget.Entry
	database  *widget.Entry
	username  *widget.Entry
	password  *widget.Entry
	masterKey *widget.Entry
	result    *widget.Label
	next      *widget.Button
	previous  *widget.Button
}

func (s ExtractSecrets) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, previous, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(BackendSelectionStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		s.result.SetText("🔵 Validating Octopus credentials.")
		s.dbServer.Disable()
		s.port.Disable()
		s.database.Disable()
		s.masterKey.Disable()
		s.next.Disable()
		s.previous.Disable()
		defer s.dbServer.Enable()
		defer s.port.Enable()
		defer s.database.Enable()
		defer s.masterKey.Enable()
		defer s.next.Enable()
		defer s.previous.Enable()

		validationFailed := false
		if err := validators.ValidateDatabase(s.getState()); err != nil {
			if err := logutil.WriteTextToFile("extract_secrets_error.txt", err.Error()); err != nil {
				fmt.Println("Failed to write error to file")
			}

			s.result.SetText("🔴 Unable to contact the database. Check the server, port, database, username, and password.")
			validationFailed = true
		}

		if !validationFailed {
			newState := s.getState()
			variableValue, err := sensitivevariables.ExtractVariables(newState.DatabaseServer, newState.DatabasePort, newState.DatabaseName, newState.DatabaseUser, newState.DatabasePass, newState.DatabaseMasterKey)

			if err != nil {
				if err := logutil.WriteTextToFile("extract_secrets_error.txt", err.Error()); err != nil {
					fmt.Println("Failed to write error to file")
				}

				s.result.SetText("🔴 Unable to extract the sensitive values. Check the server, port, database, username, password, and master key.")
				validationFailed = true
			} else {
				if err := sensitivevariables.CreateSecretsLibraryVariableSet(variableValue, newState); err != nil {
					if err := logutil.WriteTextToFile("extract_secrets_error.txt", err.Error()); err != nil {
						fmt.Println("Failed to write error to file")
					}

					s.result.SetText("🔴 Unable to create the library variable set or the variable it contains.")
					validationFailed = true
				}
			}
		}

		nexCallback := func(proceed bool) {
			if proceed {
				s.Wizard.ShowWizardStep(StepTemplateStep{
					Wizard:   s.Wizard,
					BaseStep: BaseStep{State: s.State}})
			}
		}

		if validationFailed {
			dialog.NewConfirm("Variable extraction failed", "Failed to extract the sensitive values. Do you wish to continue anyway?", nexCallback, s.Wizard.Window).Show()
		} else {
			nexCallback(true)
		}
	})
	s.next = next
	s.previous = previous
	s.result = widget.NewLabel("")

	validation := func(input string) {
		next.Disable()

		if s.dbServer.Text == "" || s.database.Text == "" || s.port.Text == "" || s.masterKey.Text == "" || s.password.Text == "" || s.username.Text == "" {
			return
		}

		next.Enable()
	}

	heading := widget.NewLabel("Sensitive Value Extraction")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	introText := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Enter the Octopus database server, port, database name, username, password, and master key.
		The master key is used to decrypt the sensitive values stored in the database.`))
	linkUrl, _ := url.Parse("https://octopus.com/docs/security/data-encryption")
	link := widget.NewHyperlink("Learn about the master key.", linkUrl)

	serverLabel := widget.NewLabel("Octopus Database Server")
	s.dbServer = widget.NewEntry()
	s.dbServer.SetPlaceHolder("192.168.1.1")
	s.dbServer.SetText(s.State.DatabaseServer)

	portLabel := widget.NewLabel("Octopus Database Port")
	s.port = widget.NewEntry()
	s.port.SetPlaceHolder("1433")
	s.port.SetText(fmt.Sprint(s.State.DatabasePort))

	databaseLabel := widget.NewLabel("Octopus Database Name")
	s.database = widget.NewEntry()
	s.database.SetPlaceHolder("Octopus")
	s.database.SetText(fmt.Sprint(s.State.DatabaseName))

	usernameLabel := widget.NewLabel("Octopus Database Username")
	s.username = widget.NewEntry()
	s.username.SetPlaceHolder("SA")
	s.username.SetText(fmt.Sprint(s.State.DatabaseUser))

	passwordLabel := widget.NewLabel("Octopus Database Password")
	s.password = widget.NewPasswordEntry()
	s.password.SetPlaceHolder("xxxxxxxxxxxxxxxxxxxxxxxxxx")
	s.password.SetText(s.State.DatabasePass)

	masterKeyPassword := widget.NewLabel("Octopus Database MasterKey")
	s.masterKey = widget.NewPasswordEntry()
	s.masterKey.SetPlaceHolder("xxxxxxxxxxxxxxxxxxxxxxxxxx")
	s.masterKey.SetText(s.State.DatabaseMasterKey)

	validation("")

	s.dbServer.OnChanged = validation
	s.port.OnChanged = validation
	s.database.OnChanged = validation
	s.username.OnChanged = validation
	s.password.OnChanged = validation
	s.masterKey.OnChanged = validation

	formLayout := container.New(layout.NewFormLayout(), serverLabel, s.dbServer, portLabel, s.port, databaseLabel, s.database, usernameLabel, s.username, passwordLabel, s.password, masterKeyPassword, s.masterKey)

	middle := container.New(layout.NewVBoxLayout(), heading, introText, link, formLayout, s.result)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s ExtractSecrets) getState() state.State {
	return state.State{
		BackendType:                   s.State.BackendType,
		Server:                        s.State.Server,
		ServerExternal:                "",
		ApiKey:                        s.State.ApiKey,
		Space:                         s.State.Space,
		DestinationServer:             s.State.DestinationServer,
		DestinationServerExternal:     "",
		DestinationApiKey:             s.State.DestinationApiKey,
		DestinationSpace:              s.State.DestinationSpace,
		AwsAccessKey:                  s.State.AwsAccessKey,
		AwsSecretKey:                  s.State.AwsSecretKey,
		AwsS3Bucket:                   s.State.AwsS3Bucket,
		AwsS3BucketRegion:             s.State.AwsS3BucketRegion,
		PromptForDelete:               s.State.PromptForDelete,
		UseContainerImages:            s.State.UseContainerImages,
		AzureResourceGroupName:        s.State.AzureResourceGroupName,
		AzureStorageAccountName:       s.State.AzureStorageAccountName,
		AzureContainerName:            s.State.AzureContainerName,
		AzureSubscriptionId:           s.State.AzureSubscriptionId,
		AzureTenantId:                 s.State.AzureTenantId,
		AzureApplicationId:            s.State.AzureApplicationId,
		AzurePassword:                 s.State.AzurePassword,
		ExcludeAllLibraryVariableSets: false,
		EnableVariableSpreading:       false,
		DatabaseServer:                strings.TrimSpace(s.dbServer.Text),
		DatabaseUser:                  strings.TrimSpace(s.username.Text),
		DatabasePass:                  strings.TrimSpace(s.password.Text),
		DatabasePort:                  strings.TrimSpace(s.port.Text),
		DatabaseName:                  strings.TrimSpace(s.database.Text),
		DatabaseMasterKey:             strings.TrimSpace(s.masterKey.Text),
	}
}

// SaveSecretsVariable creates a library variable set with a secret value containing the contents
// of a terraform.tfvars file that populates the secrets used by the exported space
func (s *ExtractSecrets) SaveSecretsVariable() error {
	return nil
}
