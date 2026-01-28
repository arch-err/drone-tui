package builds

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/arch-err/dri/internal/tui/msg"
	"github.com/arch-err/dri/internal/tui/styles"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
)

type buildItem struct {
	build *drone.Build
}

func (i buildItem) Title() string {
	return fmt.Sprintf("%s #%d %s", styles.StatusIcon(i.build.Status), i.build.Number, i.build.Message)
}

func (i buildItem) FilterValue() string {
	return fmt.Sprintf("#%d %s %s %s %s",
		i.build.Number,
		i.build.Status,
		i.build.Message,
		i.build.Event,
		i.build.Target,
	)
}

func (i buildItem) Description() string {
	parts := []string{
		i.build.Event,
		i.build.Target,
		i.build.Author,
	}
	if i.build.Finished > 0 {
		parts = append(parts, timeAgo(i.build.Finished))
	} else if i.build.Started > 0 {
		parts = append(parts, "started "+timeAgo(i.build.Started))
	}
	return strings.Join(parts, " | ")
}

// Custom delegate with compact rendering (no spacing between title and description)
type compactDelegate struct {
	list.DefaultDelegate
}

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(buildItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	var cursor string
	titleStyle := lipgloss.NewStyle()
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	if isSelected {
		cursor = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true).
			Render("> ")
		titleStyle = titleStyle.Foreground(lipgloss.Color("63")).Bold(true)
		descStyle = descStyle.Foreground(lipgloss.Color("63"))
	} else {
		cursor = "  "
	}

	title := titleStyle.Render(i.Title())
	desc := descStyle.Render(i.Description())

	// Render with cursor indicator and blank line separator
	fmt.Fprintf(w, "%s%s\n%s%s\n", cursor, title, cursor, desc)
}

func (d compactDelegate) Height() int {
	return 3 // Title line + description line + blank separator
}

func (d compactDelegate) Spacing() int {
	return 0 // No additional spacing needed, we handle it in Render
}

type Model struct {
	list          list.Model
	pendingGCount int
}

func New(buildList []*drone.Build, repoSlug string, width, height int) Model {
	items := make([]list.Item, len(buildList))
	for i, b := range buildList {
		items[i] = buildItem{build: b}
	}

	delegate := compactDelegate{}
	l := list.New(items, delegate, width, height)
	l.SetShowTitle(false) // Title shown in external statusbar instead
	l.DisableQuitKeybindings()
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)

	// Select the first item by default
	if len(items) > 0 {
		l.Select(0)
	}

	return Model{list: l}
}

func (m Model) Update(msgin tea.Msg) (Model, tea.Cmd) {
	if kmsg, ok := msgin.(tea.KeyMsg); ok {
		switch kmsg.String() {
		case "enter":
			if !m.IsFiltering() {
				if item, ok := m.list.SelectedItem().(buildItem); ok {
					return m, func() tea.Msg {
						return msg.BuildSelectedMsg{Build: item.build}
					}
				}
			}

		case "g":
			// Vim binding: gg to go to top
			if !m.IsFiltering() {
				m.pendingGCount++
				if m.pendingGCount == 2 {
					m.list.Select(0)
					m.pendingGCount = 0
				}
				return m, nil
			}

		case "G":
			// Vim binding: G to go to bottom
			if !m.IsFiltering() {
				m.list.Select(len(m.list.Items()) - 1)
				m.pendingGCount = 0
			}
			return m, nil

		default:
			// Reset pending g count on any other key
			m.pendingGCount = 0
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msgin)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

func (m Model) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m *Model) SetSize(w, h int) {
	m.list.SetSize(w, h)
}

func timeAgo(unix int64) string {
	t := time.Unix(unix, 0)
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
