package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
)

type RenameProjectStep struct {
	BaseStep
	Wizard wizard.Wizard
}

func (s RenameProjectStep) GetContainer(parent fyne.Window) *fyne.Container {

	bottom, _, _ := s.BuildNavigation(func() {
		s.Wizard.ShowWizardStep(ToolsSelectionStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	}, func() {
		s.Wizard.ShowWizardStep(BackendSelectionStep{
			Wizard:   s.Wizard,
			BaseStep: BaseStep{State: s.State}})
	})

	heading := widget.NewLabel("Rename Destination Project")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Enabling this option exposes a prompted variable defining the name of the destination project.
		This is useful if you wish to recreate the source project in the destination space multiple times with different names.
	`))

	label2 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		This option should be left disabled if you are migrating one space to another.
	`))

	label3 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		However, if you are using the projects in the source space as templates for new projects in the destination space,
		enabling this option allows you to specify the name of the destination project at runtime.
	`))

	radio := widget.NewRadioGroup([]string{"Enable Project Renaming", "Disable Project Renaming"}, func(value string) {
		s.State.EnableProjectRenaming = value == "Enable Project Renaming"
	})

	if s.State.EnableProjectRenaming {
		radio.SetSelected("Enable Project Renaming")
	} else {
		radio.SetSelected("Disable Project Renaming")
	}

	middle := container.New(layout.NewVBoxLayout(), heading, label1, label2, label3, radio)

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}
