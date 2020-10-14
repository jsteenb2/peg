package main

import (
	"errors"
	"fmt"

	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	prog := tea.NewProgram(initialize, update, view)

	if err := prog.Start(); err != nil {
		fmt.Println(err)
	}
}

func initialize() (tea.Model, tea.Cmd) {
	userInput := input.NewModel()
	userInput.CharLimit = 15
	userInput.Placeholder = "name your command"
	userInput.Focus()

	return M{
		commands: userInput,
	}, nil
}

type M struct {
	commands input.Model
	err      error
}

func update(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	m, ok := model.(M)
	if !ok {
		return M{
			err: errors.New("invalid state"),
		}, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyEnter:
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.commands, cmd = input.Update(msg, m.commands)
	return m, cmd
}

func view(model tea.Model) string {
	m, ok := model.(M)
	if !ok {
		return "invalid state view"
	}
	return input.View(m.commands)
}
