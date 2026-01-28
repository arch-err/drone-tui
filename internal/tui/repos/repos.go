package repos

import (
	"fmt"
	"strings"
	"time"

	"github.com/arch-err/dri/internal/tui/msg"
	"github.com/arch-err/dri/internal/tui/styles"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/drone/drone-go/drone"
)

type repoItem struct {
	repo *drone.Repo
}

func (i repoItem) Title() string       { return i.repo.Slug }
func (i repoItem) FilterValue() string { return i.repo.Slug }
func (i repoItem) Description() string {
	if !i.repo.Active {
		return "inactive"
	}
	b := i.repo.Build
	if b.Number == 0 {
		return "no builds"
	}
	parts := []string{
		fmt.Sprintf("%s #%d", styles.StatusIcon(b.Status), b.Number),
	}
	if b.Finished > 0 {
		parts = append(parts, timeAgo(b.Finished))
	}
	return strings.Join(parts, " · ")
}

type Model struct {
	list            list.Model
	allRepos        []*drone.Repo
	showInactive    bool
	lastEscapeAt    time.Time
	showEscapeHint  bool
	width           int
	height          int
	pendingGCount   int
}

func New(repos []*drone.Repo, width, height int) Model {
	m := Model{
		allRepos:     repos,
		showInactive: false,
		width:        width,
		height:       height,
	}
	m.rebuildList()
	// Start in filter mode by simulating "/" key press
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	m.list, _ = m.list.Update(keyMsg)
	return m
}

func (m *Model) rebuildList() {
	var items []list.Item
	for _, r := range m.allRepos {
		if !m.showInactive && !r.Active {
			continue
		}
		items = append(items, repoItem{repo: r})
	}

	m.list = list.New(items, list.NewDefaultDelegate(), m.width, m.height)
	m.list.Title = "Repositories"
	if m.showInactive {
		m.list.Title = "Repositories (showing all)"
	}
	m.list.DisableQuitKeybindings()
	m.list.SetShowStatusBar(true)
	m.list.SetFilteringEnabled(true)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msgin tea.Msg) (Model, tea.Cmd) {
	switch msgin := msgin.(type) {
	case msg.ClearEscapeHintMsg:
		m.showEscapeHint = false
		return m, nil

	case tea.KeyMsg:
		switch msgin.String() {
		case "enter":
			if !m.IsFiltering() {
				if item, ok := m.list.SelectedItem().(repoItem); ok {
					return m, func() tea.Msg {
						return msg.RepoSelectedMsg{Repo: item.repo}
					}
				}
			}

		case "a":
			// Toggle showing inactive repos
			if !m.IsFiltering() {
				m.showInactive = !m.showInactive
				m.rebuildList()
				return m, nil
			}

		case "esc":
			// Double-escape to quit
			if !m.IsFiltering() {
				now := time.Now()
				if now.Sub(m.lastEscapeAt) < 500*time.Millisecond {
					m.showEscapeHint = false
					return m, tea.Quit
				}
				m.lastEscapeAt = now
				m.showEscapeHint = true
				// Set timer to clear hint after 2 seconds
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return msg.ClearEscapeHintMsg{}
				})
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
	help := ""
	if !m.IsFiltering() {
		if m.showEscapeHint {
			// Show escape hint when user pressed escape once
			help = styles.HelpStyle.Render("Press escape again to exit")
		} else if m.showInactive {
			help = styles.HelpStyle.Render("a: hide inactive")
		} else {
			help = styles.HelpStyle.Render("a: show all · esc esc: quit")
		}
	}
	if help != "" {
		return m.list.View() + "\n\n" + help
	}
	return m.list.View()
}

func (m Model) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
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
