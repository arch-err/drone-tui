package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/arch-err/dri/internal/client"
	"github.com/arch-err/dri/internal/tui/builds"
	"github.com/arch-err/dri/internal/tui/logs"
	"github.com/arch-err/dri/internal/tui/msg"
	"github.com/arch-err/dri/internal/tui/repos"
	"github.com/arch-err/dri/internal/tui/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/drone/drone-go/drone"
)

type state int

const (
	stateLoadingRepos state = iota
	stateRepoList
	stateLoadingBuilds
	stateBuildList
	stateLoadingBuild
	stateLogViewer
)

type Model struct {
	state            state
	client           client.Client
	spinner          spinner.Model
	width            int
	height           int
	err              error
	isRefreshing     bool
	loadingStartTime time.Time

	repoList  repos.Model
	buildList builds.Model
	logViewer logs.Model

	selectedRepo  *drone.Repo
	selectedBuild *drone.Build

	// Pending data waiting for minimum loading time
	pendingRepos  []*drone.Repo
	pendingBuilds []*drone.Build
	pendingBuild  *drone.Build
}

const minLoadingDuration = 500 * time.Millisecond

type loadingCompleteMsg struct{}

func New(c client.Client) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return Model{
		state:            stateLoadingRepos,
		client:           c,
		spinner:          s,
		loadingStartTime: time.Now(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadReposCmd())
}

func (m Model) Update(teaMsg tea.Msg) (tea.Model, tea.Cmd) {
	switch teaMsg := teaMsg.(type) {
	case tea.WindowSizeMsg:
		m.width = teaMsg.Width
		m.height = teaMsg.Height
		return m.propagateSize(), nil

	case tea.KeyMsg:
		if key.Matches(teaMsg, keys.Quit) {
			if m.state == stateRepoList && m.repoList.IsFiltering() {
				break
			}
			if m.state == stateBuildList && m.buildList.IsFiltering() {
				break
			}
			return m, tea.Quit
		}

		// Refresh keybind
		if teaMsg.String() == "r" {
			switch m.state {
			case stateRepoList:
				if !m.repoList.IsFiltering() {
					m.state = stateLoadingRepos
					m.isRefreshing = true
					m.loadingStartTime = time.Now()
					return m, tea.Batch(m.spinner.Tick, m.loadReposCmd())
				}
			case stateBuildList:
				if !m.buildList.IsFiltering() {
					m.state = stateLoadingBuilds
					m.isRefreshing = true
					m.loadingStartTime = time.Now()
					return m, tea.Batch(m.spinner.Tick, m.loadBuildsCmd(m.selectedRepo.Namespace, m.selectedRepo.Name))
				}
			case stateLogViewer:
				m.state = stateLoadingBuild
				m.isRefreshing = true
				m.loadingStartTime = time.Now()
				return m, tea.Batch(m.spinner.Tick, m.loadBuildCmd(m.selectedRepo.Namespace, m.selectedRepo.Name, int(m.selectedBuild.Number)))
			}
		}

	case msg.ReposLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			return m, tea.Quit
		}
		elapsed := time.Since(m.loadingStartTime)
		if elapsed < minLoadingDuration {
			m.pendingRepos = teaMsg.Repos
			return m, tea.Tick(minLoadingDuration-elapsed, func(t time.Time) tea.Msg {
				return loadingCompleteMsg{}
			})
		}
		m.repoList = repos.New(teaMsg.Repos, m.width, m.height)
		m.state = stateRepoList
		m.isRefreshing = false
		return m, nil

	case msg.RepoSelectedMsg:
		m.selectedRepo = teaMsg.Repo
		m.state = stateLoadingBuilds
		m.loadingStartTime = time.Now()
		return m, tea.Batch(m.spinner.Tick, m.loadBuildsCmd(teaMsg.Repo.Namespace, teaMsg.Repo.Name))

	case msg.BuildsLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			m.state = stateRepoList
			m.isRefreshing = false
			return m, nil
		}
		elapsed := time.Since(m.loadingStartTime)
		if elapsed < minLoadingDuration {
			m.pendingBuilds = teaMsg.Builds
			return m, tea.Tick(minLoadingDuration-elapsed, func(t time.Time) tea.Msg {
				return loadingCompleteMsg{}
			})
		}
		// Account for statusbar height
		m.buildList = builds.New(teaMsg.Builds, m.selectedRepo.Slug, m.width, m.height-1)
		m.state = stateBuildList
		m.isRefreshing = false
		return m, nil

	case msg.BuildSelectedMsg:
		m.selectedBuild = teaMsg.Build
		m.state = stateLoadingBuild
		m.loadingStartTime = time.Now()
		return m, tea.Batch(m.spinner.Tick, m.loadBuildCmd(m.selectedRepo.Namespace, m.selectedRepo.Name, int(teaMsg.Build.Number)))

	case msg.BuildLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			m.state = stateBuildList
			m.isRefreshing = false
			return m, nil
		}
		elapsed := time.Since(m.loadingStartTime)
		if elapsed < minLoadingDuration {
			m.pendingBuild = teaMsg.Build
			return m, tea.Tick(minLoadingDuration-elapsed, func(t time.Time) tea.Msg {
				return loadingCompleteMsg{}
			})
		}
		m.selectedBuild = teaMsg.Build
		// Account for statusbar height
		m.logViewer = logs.New(teaMsg.Build, m.width, m.height-1)
		m.state = stateLogViewer
		m.isRefreshing = false
		return m, m.loadAllLogsCmd(teaMsg.Build)

	case loadingCompleteMsg:
		m.isRefreshing = false
		switch m.state {
		case stateLoadingRepos:
			if m.pendingRepos != nil {
				m.repoList = repos.New(m.pendingRepos, m.width, m.height)
				m.pendingRepos = nil
				m.state = stateRepoList
			}
		case stateLoadingBuilds:
			if m.pendingBuilds != nil {
				m.buildList = builds.New(m.pendingBuilds, m.selectedRepo.Slug, m.width, m.height-1)
				m.pendingBuilds = nil
				m.state = stateBuildList
			}
		case stateLoadingBuild:
			if m.pendingBuild != nil {
				m.selectedBuild = m.pendingBuild
				m.logViewer = logs.New(m.pendingBuild, m.width, m.height-1)
				m.pendingBuild = nil
				m.state = stateLogViewer
				return m, m.loadAllLogsCmd(m.selectedBuild)
			}
		}
		return m, nil
	}

	var cmd tea.Cmd

	switch m.state {
	case stateLoadingRepos, stateLoadingBuilds, stateLoadingBuild:
		m.spinner, cmd = m.spinner.Update(teaMsg)
		return m, cmd

	case stateRepoList:
		var repoCmd tea.Cmd
		m.repoList, repoCmd = m.repoList.Update(teaMsg)
		return m, repoCmd

	case stateBuildList:
		if kmsg, ok := teaMsg.(tea.KeyMsg); ok && key.Matches(kmsg, keys.Back) {
			if !m.buildList.IsFiltering() {
				m.state = stateRepoList
				return m, nil
			}
		}
		var buildCmd tea.Cmd
		m.buildList, buildCmd = m.buildList.Update(teaMsg)
		return m, buildCmd

	case stateLogViewer:
		if kmsg, ok := teaMsg.(tea.KeyMsg); ok && key.Matches(kmsg, keys.Back) {
			m.state = stateBuildList
			return m, nil
		}
		var logCmd tea.Cmd
		m.logViewer, logCmd = m.logViewer.Update(teaMsg)
		return m, logCmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return styles.AppStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit.", m.err))
	}

	statusBar := m.renderStatusBar()

	switch m.state {
	case stateLoadingRepos:
		// Show repo list while refreshing, or just statusbar on initial load
		if m.isRefreshing {
			return m.repoList.View()
		}
		if statusBar != "" {
			return statusBar
		}
		return styles.AppStyle.Render(m.spinner.View() + " Loading repositories...")

	case stateRepoList:
		return m.repoList.View()

	case stateLoadingBuilds:
		// Show build list while refreshing, or repo list on initial navigation
		if m.isRefreshing {
			if statusBar != "" {
				return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.buildList.View())
			}
			return m.buildList.View()
		}
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.repoList.View())
		}
		return m.repoList.View()

	case stateBuildList:
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.buildList.View())
		}
		return m.buildList.View()

	case stateLoadingBuild:
		// Show log viewer while refreshing, or build list on initial navigation
		if m.isRefreshing {
			if statusBar != "" {
				return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.logViewer.View())
			}
			return m.logViewer.View()
		}
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.buildList.View())
		}
		return m.buildList.View()

	case stateLogViewer:
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.logViewer.View())
		}
		return m.logViewer.View()
	}

	return ""
}

func (m Model) renderStatusBar() string {
	var parts []string
	var loadingText string

	statusBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1)

	loadingStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("244")).
		Padding(0, 1)

	switch m.state {
	case stateLoadingRepos:
		loadingText = "● Refreshing..."
		parts = append(parts, loadingStyle.Render(loadingText))

	case stateRepoList:
		// No statusbar for repo list
		return ""

	case stateLoadingBuilds:
		if m.selectedRepo != nil {
			parts = append(parts, highlightStyle.Render(m.selectedRepo.Slug))
		}
		loadingText = "● Refreshing..."

	case stateBuildList:
		if m.selectedRepo != nil {
			parts = append(parts, highlightStyle.Render(m.selectedRepo.Slug))
		}

	case stateLoadingBuild:
		if m.selectedRepo != nil {
			parts = append(parts, highlightStyle.Render(m.selectedRepo.Slug))
		}
		loadingText = "● Refreshing..."

	case stateLogViewer:
		if m.selectedRepo != nil {
			parts = append(parts, highlightStyle.Render(m.selectedRepo.Slug))
		}
		if m.selectedBuild != nil {
			// Truncate commit message to 12 chars and strip newlines
			msg := strings.ReplaceAll(m.selectedBuild.Message, "\n", " ")
			msg = strings.ReplaceAll(msg, "\r", " ")
			if len(msg) > 12 {
				msg = msg[:12] + "..."
			}
			parts = append(parts, statusBarStyle.Render(msg))
		}
		// Add log tabs to statusbar
		parts = append(parts, m.logViewer.RenderStatusBar())
	}

	if len(parts) == 0 && loadingText == "" {
		return ""
	}

	// Join all parts - they already have their own backgrounds
	joined := lipgloss.JoinHorizontal(lipgloss.Top, parts...)

	// Fill remaining width with background color, add loading on the right
	if m.width > 0 {
		contentWidth := lipgloss.Width(joined)
		loadingWidth := 0
		if loadingText != "" {
			loadingWidth = lipgloss.Width(loadingStyle.Render(loadingText))
		}
		fillWidth := m.width - contentWidth - loadingWidth
		fillStyle := lipgloss.NewStyle().Background(lipgloss.Color("235"))
		if fillWidth > 0 {
			joined = joined + fillStyle.Render(strings.Repeat(" ", fillWidth))
		}
		if loadingText != "" {
			joined = joined + loadingStyle.Render(loadingText)
		}
	}
	return joined
}

func (m *Model) propagateSize() Model {
	switch m.state {
	case stateRepoList:
		m.repoList.SetSize(m.width, m.height)
	case stateBuildList:
		m.buildList.SetSize(m.width, m.height-1) // Account for statusbar
	case stateLogViewer:
		m.logViewer.SetSize(m.width, m.height-1) // Account for statusbar
	}
	return *m
}

func (m Model) loadReposCmd() tea.Cmd {
	return func() tea.Msg {
		repoList, err := m.client.ListRepos()
		return msg.ReposLoadedMsg{Repos: repoList, Err: err}
	}
}

func (m Model) loadBuildsCmd(namespace, name string) tea.Cmd {
	return func() tea.Msg {
		buildList, err := m.client.ListBuilds(namespace, name, 1)
		return msg.BuildsLoadedMsg{Builds: buildList, Err: err}
	}
}

func (m Model) loadBuildCmd(namespace, name string, number int) tea.Cmd {
	return func() tea.Msg {
		build, err := m.client.GetBuild(namespace, name, number)
		return msg.BuildLoadedMsg{Build: build, Err: err}
	}
}

func (m Model) loadAllLogsCmd(build *drone.Build) tea.Cmd {
	var cmds []tea.Cmd
	for _, stage := range build.Stages {
		for _, step := range stage.Steps {
			s := stage
			st := step
			cmds = append(cmds, func() tea.Msg {
				lines, err := m.client.GetLogs(
					m.selectedRepo.Namespace,
					m.selectedRepo.Name,
					int(build.Number),
					int(s.Number),
					int(st.Number),
				)
				return msg.LogsLoadedMsg{
					StepName: st.Name,
					StageNum: int(s.Number),
					StepNum:  int(st.Number),
					Lines:    lines,
					Err:      err,
				}
			})
		}
	}
	return tea.Batch(cmds...)
}
