package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
)

type EnableVariableSpreading struct {
	BaseStep
	Wizard wizard.Wizard
}

func (s EnableVariableSpreading) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, _, _ := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(BackendSelectionStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		if s.State.EnableVariableSpreading {
			s.Wizard.ShowWizardStep(SpreadVariablesStep{
				Wizard:   s.Wizard,
				BaseStep: BaseStep{State: s.State}})
		} else {
			s.Wizard.ShowWizardStep(StepTemplateStep{
				Wizard:   s.Wizard,
				BaseStep: BaseStep{State: s.State}})
		}
	})

	heading := widget.NewLabel("Enable Variable Spreading")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Do you wish to enable variable spreading?
		Variable spreading modifies sensitive variables to allow them to be automatically migrated to the destination server.
		The modifications are irreversible and have security implications.
		Only enable variable spreading if you understand the implications.
		Disabling variable spreading means sensitive variables are migrated with dummy values and must be manually updated on the destination server.
	`))

	radio := widget.NewRadioGroup([]string{"Enable variable spreading", "Disable variable spreading"}, func(value string) {
		s.State.EnableVariableSpreading = value == "Enable variable spreading"
	})

	if s.State.EnableVariableSpreading {
		radio.SetSelected("Enable variable spreading")
	} else {
		radio.SetSelected("Disable variable spreading")
	}

	middle := container.New(layout.NewVBoxLayout(), heading, label1, radio)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}
