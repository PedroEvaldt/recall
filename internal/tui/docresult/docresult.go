package docresult

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Choices contains the overwrite options shown when a config file already exists.
var Choices = []string{"No (Keep my config.yaml)", "Yes (Delete my config.yaml)"}

// ResultModel stores the state for the config overwrite confirmation prompt.
type ResultModel struct {
	cursor int
	Choice string
}

// Init does not need an initial command for the confirmation prompt.
func (m ResultModel) Init() tea.Cmd {
	return nil
}

// Update handles navigation and saves the selected choice on enter.
func (m ResultModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
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

// View renders the overwrite confirmation prompt.
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
