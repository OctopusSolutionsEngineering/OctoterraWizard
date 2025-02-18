package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
)

type AdvancedOptionsStep struct {
	BaseStep
	Wizard wizard.Wizard
}

func (s AdvancedOptionsStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, _, _ := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(BackendSelectionStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		s.Wizard.ShowWizardStep(SpreadVariablesStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	})

	heading := widget.NewLabel("Advanced Options")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		These are advanced options to customize the migration process.
		Leave these options as default unless you have a specific reason to change them.
	`))

	radio := widget.NewRadioGroup([]string{"Export all library variable sets", "Exclude all library variable sets"}, func(value string) {
		s.State.ExcludeAllLibraryVariableSets = value == "Exclude all library variable sets"
	})

	if s.State.ExcludeAllLibraryVariableSets {
		radio.SetSelected("Exclude all library variable sets")
	} else {
		radio.SetSelected("Export all library variable sets")
	}

	middle := container.New(layout.NewVBoxLayout(), heading, label1, radio)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}
