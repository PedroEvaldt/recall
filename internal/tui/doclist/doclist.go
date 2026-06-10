package doclist

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/PedroEvaldt/recall/internal/api"
	"github.com/PedroEvaldt/recall/internal/client"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type docItem struct {
	doc api.DocumentResponse
}

type documentsListMsg struct {
	docs []api.DocumentResponse
}

type errMsg struct {
	err error
}

func descriptionConverter(mime string, tags []string) string {
	s := "Type: " + mime + "\n"
	s += strings.Join(tags, ", ")
	return s
}

func (d docItem) Title() string       { return d.doc.Title }
func (d docItem) Description() string { return descriptionConverter(d.doc.MimeType, d.doc.Tags) }
func (d docItem) FilterValue() string { return d.doc.Title }

// Model stores the document list state for the Bubble Tea runtime.
type Model struct {
	Aborted  bool
	query    string
	limit    string
	ctx      context.Context
	list     list.Model
	c        *client.Client
	Selected *api.DocumentResponse
}

// New creates a document list model that fetches results with the provided client.
func New(ctx context.Context, c *client.Client, query, limit string) Model {
	delegate := list.NewDefaultDelegate()
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = "Buscando: " + query
	return Model{
		ctx:   ctx,
		list:  l,
		c:     c,
		query: query,
		limit: limit,
	}
}

// Init starts the initial document fetch command.
func (m Model) Init() tea.Cmd {
	return m.fetchDocs
}

func (m Model) fetchDocs() tea.Msg {
	docs, err := m.c.ListDocuments(m.ctx, m.query, m.limit)
	if err != nil {
		return errMsg{err}
	}
	return documentsListMsg{docs}
}

// Update handles result loading, keyboard selection, cancellation, and resize events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case documentsListMsg:
		items := make([]list.Item, 0, len(msg.docs))
		for _, doc := range msg.docs {
			items = append(items, docItem{doc: doc})
		}

		m.list.SetItems(items)
		m.list.Title = "Results for: " + m.query
		return m, nil

	case errMsg:
		m.list.Title = "Erro: " + msg.err.Error()
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Aborted = true
			return m, tea.Quit
		case "enter":
			if m.list.FilterState() == list.Filtering {
				break
			}
			if item, ok := m.list.SelectedItem().(docItem); ok {
				m.Selected = &item.doc
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the list in the terminal alternate screen.
func (m Model) View() tea.View {
	v := tea.NewView(docStyle.Render(m.list.View()))
	v.AltScreen = true
	return v
}
