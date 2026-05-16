package docresult

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

var Choices = []string{"No (Keep my config.yaml)", "Yes (Delete my config.yaml)"}

type ResultModel struct {
	cursor int
	Choice string
}

func (m ResultModel) Init() tea.Cmd {
	return nil
}

func (m ResultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the Choice on the channel and exit.
			m.Choice = Choices[m.cursor]
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(Choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(Choices) - 1
			}
		}
	}

	return m, nil
}

func (m ResultModel) View() tea.View {
	s := strings.Builder{}
	s.WriteString("You already have a config.yaml file, would you like to replace it?\n\n")

	for i := range Choices {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(Choices[i])
		s.WriteString("\n")
	}
	s.WriteString("\n(press q to quit)\n")

	return tea.NewView(s.String())
}
