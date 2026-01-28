package logs

import (
	"fmt"
	"strings"

	"github.com/arch-err/dri/internal/tui/msg"
	"github.com/arch-err/dri/internal/tui/styles"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
)

type stepTab struct {
	name     string
	stageNum int
	stepNum  int
	content  string
	loaded   bool
}

type Model struct {
	tabs          []stepTab
	activeTab     int
	viewport      viewport.Model
	spinner       spinner.Model
	width         int
	height        int
	buildNum      int64
	pendingGCount int
}

func New(build *drone.Build, width, height int) Model {
	var tabs []stepTab
	for _, stage := range build.Stages {
		for _, step := range stage.Steps {
			tabs = append(tabs, stepTab{
				name:     step.Name,
				stageNum: int(stage.Number),
				stepNum:  int(step.Number),
			})
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	vp := viewport.New(width, height-2) // Reduced from -4 for compact layout

	m := Model{
		tabs:     tabs,
		viewport: vp,
		spinner:  s,
		width:    width,
		height:   height,
		buildNum: build.Number,
	}

	if len(tabs) > 0 {
		m.viewport.SetContent("Loading...")
	}

	return m
}

func (m Model) Update(msgin tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msgin := msgin.(type) {
	case tea.KeyMsg:
		switch msgin.String() {
		case "tab":
			if len(m.tabs) > 0 {
				m.activeTab = (m.activeTab + 1) % len(m.tabs)
				m.updateViewportContent()
			}
			return m, nil

		case "shift+tab":
			if len(m.tabs) > 0 {
				m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
				m.updateViewportContent()
			}
			return m, nil

		case "g":
			// Vim binding: gg to go to top
			m.pendingGCount++
			if m.pendingGCount == 2 {
				m.viewport.GotoTop()
				m.pendingGCount = 0
			}
			return m, nil

		case "G":
			// Vim binding: G to go to bottom
			m.viewport.GotoBottom()
			m.pendingGCount = 0
			return m, nil

		default:
			// Reset pending g count on any other key
			m.pendingGCount = 0
		}

	case msg.LogsLoadedMsg:
		for i, tab := range m.tabs {
			if tab.stageNum == msgin.StageNum && tab.stepNum == msgin.StepNum {
				if msgin.Err != nil {
					m.tabs[i].content = fmt.Sprintf("Error loading logs: %v", msgin.Err)
				} else {
					var lines []string
					for _, line := range msgin.Lines {
						// Trim trailing whitespace/newlines from each line
						lines = append(lines, strings.TrimRight(line.Message, "\n\r"))
					}
					// Join with single newlines (no extra spacing)
					m.tabs[i].content = strings.Join(lines, "\n")
				}
				m.tabs[i].loaded = true
				if i == m.activeTab {
					m.updateViewportContent()
				}
				break
			}
		}
		return m, nil
	}

	var spinCmd tea.Cmd
	m.spinner, spinCmd = m.spinner.Update(msgin)
	cmds = append(cmds, spinCmd)

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msgin)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateViewportContent() {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		tab := m.tabs[m.activeTab]
		if tab.loaded {
			m.viewport.SetContent(tab.content)
		} else {
			m.viewport.SetContent(m.spinner.View() + " Loading...")
		}
		m.viewport.GotoTop()
	}
}

func (m Model) View() string {
	if len(m.tabs) == 0 {
		return styles.AppStyle.Render("No steps found in this build.")
	}

	return m.viewport.View()
}

// RenderStatusBar renders the tab bar as a single line statusbar
func (m Model) RenderStatusBar() string {
	if len(m.tabs) == 0 {
		return ""
	}

	var parts []string
	for i, tab := range m.tabs {
		label := tab.name
		if !tab.loaded {
			label += " " + m.spinner.View()
		}

		style := lipgloss.NewStyle().Padding(0, 1)
		if i == m.activeTab {
			style = style.Background(lipgloss.Color("63")).Foreground(lipgloss.Color("231")).Bold(true)
		} else {
			style = style.Foreground(lipgloss.Color("244"))
		}

		parts = append(parts, style.Render(label))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h
	m.viewport.Width = w
	m.viewport.Height = h - 2 // Compact layout
}
