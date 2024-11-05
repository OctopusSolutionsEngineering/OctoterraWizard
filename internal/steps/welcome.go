package steps

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/mcasperson/OctoterraWizard/internal/strutil"
	"github.com/mcasperson/OctoterraWizard/internal/wizard"
	"net/url"
)

type WelcomeStep struct {
	BaseStep
	Wizard wizard.Wizard
}

func (s WelcomeStep) GetContainer(parent fyne.Window) *fyne.Container {

	heading := widget.NewLabel("Welcome")
	heading.TextStyle = fyne.TextStyle{Bold: true}

	label1 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Welcome to the Octoterra Wizard.
		This tool prepares your Octopus space to export it to another instance.
	`))
	label2 := widget.NewLabel(strutil.TrimMultilineWhitespace(`
		Before you continue, you MUST backup your Octopus instance.
		This tool modifies your Octopus instance and can cause data loss.
	`))
	linkUrl, _ := url.Parse("https://octopus.com/docs/administration/migrate-spaces-with-octoterra")
	link := widget.NewHyperlink("Read the documentation.", linkUrl)
	middle := container.New(layout.NewVBoxLayout(), heading, label1, label2, link)

	bottom, previous, _ := s.BuildNavigation(func() {}, func() {
		s.Wizard.ShowWizardStep(TestTerraformStep{Wizard: s.Wizard, BaseStep: BaseStep{State: s.State}})
	})
	previous.Disable()

	content := container.NewBorder(nil, bottom, nil, nil, middle)

	return content
}
