package steps

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/logutil"
	"github.com/mcasperson/OctoterraWizard/internal/spreadvariables"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"net/url"
)

type SpreadVariablesStep struct {
	BaseStep
	Wizard          wizard.Wizard
	spreadVariables *widget.Button
	confirmChanges  *widget.Check
	exportDone      bool
}

func (s SpreadVariablesStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, previous, next := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(EnableVariableSpreading{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		moveNext := func(proceed bool) {
			if !proceed {
				return
			}

			s.Wizard.ShowWizardStep(StepTemplateStep{
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
	s.exportDone = false

	heading := widget.NewLabel("Spread Variables")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	intro := widget.NewLabel(strutil.TrimMultilineWhitespace(`In order to allow sensitive variables to be exported to a new space, all sensitive variables must have a unique name and no scopes.`))
	intro.Wrapping = fyne.TextWrapWord
	intro2 := widget.NewLabel(strutil.TrimMultilineWhitespace(`However, it is common to have sensitive variables that share a name and have different scopes. A common example is a database connection string where multiple variables are called "ConnectionString" but are uniquely scoped to an individual environment.`))
	intro2.Wrapping = fyne.TextWrapWord
	intro3 := widget.NewLabel(strutil.TrimMultilineWhitespace(`This step will rename any scoped sensitive variables and remove their scopes. It will also create a regular variable with the original sensitive variable name and scopes, and refer to the renamed sensitive variable as an octostache template. This process is called "spreading" the sensitive variables.`))
	intro3.Wrapping = fyne.TextWrapWord
	intro4 := widget.NewLabel(strutil.TrimMultilineWhitespace(`Modifying variables in this way means steps can continue to refer to the original sensitive variable name, so no changes are required to the deployment process. However, removing the scopes from the sensitive variables does have security implications. In particular, all sensitive variables are exposed to all deployments and runbook runs.`))
	intro4.Wrapping = fyne.TextWrapWord

	linkUrl, _ := url.Parse("https://octopus.com/docs/administration/migrate-spaces-with-octoterra#spreading-sensitive-variables")
	link := widget.NewHyperlink("Learn more about spreading sensitive variables.", linkUrl)

	s.confirmChanges = widget.NewCheck("I understand and accept the security risks associated with spreading sensitive variables", func(value bool) {
		if value {
			s.spreadVariables.Enable()
		} else {
			s.spreadVariables.Disable()
		}
	})

	infinite := widget.NewProgressBarInfinite()
	infinite.Start()
	infinite.Hide()
	result := widget.NewLabel("")
	s.spreadVariables = widget.NewButton("Spread Sensitive Variables (click the checkbox above to continue)", func() {
		next.Disable()
		previous.Disable()
		infinite.Show()
		s.confirmChanges.Disable()
		s.spreadVariables.Disable()
		result.SetText("🔵 Spreading sensitive variables. This can take a little while.")
		s.exportDone = true

		go func() {
			defer previous.Enable()
			defer infinite.Hide()
			if err := s.Execute(); err != nil {
				if err := logutil.WriteTextToFile("spread_variables_error.txt", err.Error()); err != nil {
					fmt.Println("Failed to write error to file")
				}

				result.SetText("🔴 An error was raised while attempting to spread the variables. Unfortunately, this means the wizard can not continue.\n " + err.Error())
			} else {
				result.SetText("🟢 Sensitive variables have been spread.")
				next.Enable()
			}
		}()
	})
	s.spreadVariables.Disable()
	middle := container.New(layout.NewVBoxLayout(), heading, intro, intro2, intro3, intro4, link, s.confirmChanges, s.spreadVariables, infinite, result)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}

func (s SpreadVariablesStep) Execute() error {
	spreader := spreadvariables.VariableSpreader{
		State: s.State,
	}
	return spreader.SpreadAllVariables()
}
