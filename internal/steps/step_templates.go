package steps

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/octoclient"
	"github.com/mcasperson/OctoterraWizard/internal/query"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
)

type StepTemplateStep struct {
	BaseStep
	Wizard       wizard.Wizard
	result       *widget.Label
	logs         *widget.Entry
	exportDone   bool
	installSteps *widget.Button
}

func (s StepTemplateStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, previous, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(ExtractSecrets{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		moveNext := func(proceed bool) {
			if !proceed {
				return
			}

			s.Wizard.ShowWizardStep(PromptRemovalStep{
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

	heading := widget.NewLabel("Install Step Templates")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		The runbooks created by this wizard require a number of step templates to be installed from the community step template library.
	`))
	s.result = widget.NewLabel("")
	s.logs = widget.NewEntry()
	s.logs.Disable()
	s.logs.MultiLine = true
	s.logs.SetMinRowsVisible(20)
	s.logs.Hide()
	s.exportDone = false

	s.installSteps = widget.NewButton("Install Step Templates", func() {
		s.logs.Hide()
		s.result.SetText("🔵 Installing step templates.")
		s.exportDone = true
		previous.Disable()
		next.Disable()
		s.installSteps.Disable()
		defer previous.Enable()
		defer next.Enable()
		defer s.installSteps.Enable()

		message, err := s.Execute()
		if err != nil {
			s.result.SetText(message)
			s.logs.SetText(err.Error())
			return
		}

		next.Enable()
		s.result.SetText("🟢 Step templates installed.")
	})
	middle := container.New(layout.NewVBoxLayout(), heading, label1, s.installSteps, s.result, s.logs)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s StepTemplateStep) Execute() (string, error) {
	myclient, err := octoclient.CreateClient(s.State)

	if err != nil {
		return "🔴 Failed to create the client", err
	}

	// Octopus - Serialize Space to Terraform
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/e03c56a4-f660-48f6-9d09-df07e1ac90bd"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Serialize Project to Terraform
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/e9526501-09d5-490f-ac3f-5079735fe041"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Populate Octoterra Space (S3 Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/14d51af4-1c3d-4d41-9044-4304111d0cd8"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Populate Octoterra Space (Azure Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/c15be981-3138-47c8-a935-ab388b7840be"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Add Runbook to Project (Azure Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/9b206752-5a8c-40dd-84a8-94f08a42955c"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Add Runbook to Project (S3 Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/8b8b0386-78f8-42c2-baea-2fdb9a57c32d"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Create Octoterra Space (Azure Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/c9c5a6a2-0ce7-4d7a-8eb5-111ac44df24e"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Create Octoterra Space (S3 Backend)
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/90a8dd76-6456-49f9-9c03-baf85442aa57"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	// Octopus - Lookup Space ID
	if err, message := query.InstallStepTemplate(myclient, s.State, "https://library.octopus.com/step-templates/324f747e-e2cd-439d-a660-774baf4991f2"); err != nil {
		return message, errors.Join(errors.New("failed to install step template"), err)
	}

	return "🟢 Step templates installed.", nil
}
