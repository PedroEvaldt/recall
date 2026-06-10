package docglamour

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

var helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render

// Example is the Bubble Tea model that renders Markdown in a scrollable viewport.
type Example struct {
	viewport viewport.Model
}

// NewExample renders Markdown content into a scrollable terminal viewport.
func NewExample(content string, isDark bool) (*Example, error) {
	const (
		width  = 78
		height = 20
	)

	vp := viewport.New()
	vp.SetWidth(width)
	vp.SetHeight(height)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		PaddingRight(2)

	// We need to adjust the width of the glamour render from our main width
	// to account for a few things:
	//
	//  * The viewport border width
	//  * The viewport padding
	//  * The viewport margins
	//  * The gutter glamour applies to the left side of the content
	//
	const glamourGutter = 3
	glamourRenderWidth := width - vp.Style.GetHorizontalFrameSize() - glamourGutter

	style := styles.DarkStyleConfig
	if !isDark {
		style = styles.LightStyleConfig
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(glamourRenderWidth),
	)
	if err != nil {
		return nil, err
	}

	str, err := renderer.Render(content)
	if err != nil {
		return nil, err
	}

	vp.SetContent(str)

	return &Example{
		viewport: vp,
	}, nil
}

// Init returns nil because the viewport does not need an initial command.
func (e Example) Init() tea.Cmd {
	return nil
}

// Update handles viewport navigation and quit keys.
func (e Example) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return e, tea.Quit
		default:
			var cmd tea.Cmd
			e.viewport, cmd = e.viewport.Update(msg)
			return e, cmd
		}
	default:
		return e, nil
	}
}

// View renders the Markdown viewport and navigation help.
func (e Example) View() tea.View {
	return tea.NewView(e.viewport.View() + e.helpView())
}

func (e Example) helpView() string {
	return helpStyle("\n  ↑/↓: Navigate • q: Quit\n")
}
