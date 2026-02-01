package msg

import "github.com/drone/drone-go/drone"

type ReposLoadedMsg struct {
	Repos []*drone.Repo
	Err   error
}

type BuildsLoadedMsg struct {
	Builds []*drone.Build
	Err    error
}

type BuildLoadedMsg struct {
	Build *drone.Build
	Err   error
}

type LogsLoadedMsg struct {
	StepName string
	StageNum int
	StepNum  int
	Lines    []*drone.Line
	Err      error
}

type RepoSelectedMsg struct {
	Repo *drone.Repo
}

type BuildSelectedMsg struct {
	Build *drone.Build
}

type ClearEscapeHintMsg struct{}

type OpenBrowserMsg struct {
	URL string
}
