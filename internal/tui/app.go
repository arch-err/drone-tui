package tui

import (
	"fmt"

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
	state   state
	client  client.Client
	spinner spinner.Model
	width   int
	height  int
	err     error

	repoList  repos.Model
	buildList builds.Model
	logViewer logs.Model

	selectedRepo  *drone.Repo
	selectedBuild *drone.Build
}

func New(c client.Client) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return Model{
		state:   stateLoadingRepos,
		client:  c,
		spinner: s,
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

	case msg.ReposLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			return m, tea.Quit
		}
		m.repoList = repos.New(teaMsg.Repos, m.width, m.height)
		m.state = stateRepoList
		return m, nil

	case msg.RepoSelectedMsg:
		m.selectedRepo = teaMsg.Repo
		m.state = stateLoadingBuilds
		return m, tea.Batch(m.spinner.Tick, m.loadBuildsCmd(teaMsg.Repo.Namespace, teaMsg.Repo.Name))

	case msg.BuildsLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			m.state = stateRepoList
			return m, nil
		}
		// Account for statusbar height
		m.buildList = builds.New(teaMsg.Builds, m.selectedRepo.Slug, m.width, m.height-1)
		m.state = stateBuildList
		return m, nil

	case msg.BuildSelectedMsg:
		m.selectedBuild = teaMsg.Build
		m.state = stateLoadingBuild
		return m, tea.Batch(m.spinner.Tick, m.loadBuildCmd(m.selectedRepo.Namespace, m.selectedRepo.Name, int(teaMsg.Build.Number)))

	case msg.BuildLoadedMsg:
		if teaMsg.Err != nil {
			m.err = teaMsg.Err
			m.state = stateBuildList
			return m, nil
		}
		m.selectedBuild = teaMsg.Build
		// Account for statusbar height
		m.logViewer = logs.New(teaMsg.Build, m.width, m.height-1)
		m.state = stateLogViewer
		return m, m.loadAllLogsCmd(teaMsg.Build)
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
		return styles.AppStyle.Render(m.spinner.View() + " Loading repositories...")
	case stateRepoList:
		return m.repoList.View()
	case stateLoadingBuilds:
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, styles.AppStyle.Render(m.spinner.View()+" Loading builds..."))
		}
		return styles.AppStyle.Render(m.spinner.View() + " Loading builds...")
	case stateBuildList:
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, m.buildList.View())
		}
		return m.buildList.View()
	case stateLoadingBuild:
		if statusBar != "" {
			return lipgloss.JoinVertical(lipgloss.Left, statusBar, styles.AppStyle.Render(m.spinner.View()+" Loading build details..."))
		}
		return styles.AppStyle.Render(m.spinner.View() + " Loading build details...")
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

	statusBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1)

	switch m.state {
	case stateBuildList, stateLoadingBuild, stateLogViewer:
		if m.selectedRepo != nil {
			parts = append(parts, highlightStyle.Render(m.selectedRepo.Slug))
		}

	case stateLoadingRepos, stateRepoList:
		// No statusbar for repo list
		return ""
	}

	if m.state == stateLogViewer {
		if m.selectedBuild != nil {
			parts = append(parts, statusBarStyle.Render(fmt.Sprintf("#%d", m.selectedBuild.Number)))
		}
		// Add log tabs to statusbar
		parts = append(parts, m.logViewer.RenderStatusBar())
	}

	if len(parts) == 0 {
		return ""
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
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
