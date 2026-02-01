package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/drone/drone-go/drone"
	"golang.org/x/oauth2"
)

type Client interface {
	ListRepos() ([]*drone.Repo, error)
	ListBuilds(namespace, name string, page int) ([]*drone.Build, error)
	GetBuild(namespace, name string, number int) (*drone.Build, error)
	GetLogs(owner, name string, build, stage, step int) ([]*drone.Line, error)
	ServerURL() string
}

type droneClient struct {
	inner      drone.Client
	httpClient *http.Client
	server     string
}

func New(server, token string) Client {
	conf := new(oauth2.Config)
	auth := conf.Client(context.Background(), &oauth2.Token{AccessToken: token})
	return &droneClient{
		inner:      drone.NewClient(server, auth),
		httpClient: auth,
		server:     server,
	}
}

func (c *droneClient) ListRepos() ([]*drone.Repo, error) {
	uri := fmt.Sprintf("%s/api/user/repos?latest=true", c.server)
	resp, err := c.httpClient.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.inner.RepoList()
	}

	var repos []*drone.Repo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func (c *droneClient) ListBuilds(namespace, name string, page int) ([]*drone.Build, error) {
	return c.inner.BuildList(namespace, name, drone.ListOptions{Page: page})
}

func (c *droneClient) GetBuild(namespace, name string, number int) (*drone.Build, error) {
	return c.inner.Build(namespace, name, number)
}

func (c *droneClient) GetLogs(owner, name string, build, stage, step int) ([]*drone.Line, error) {
	return c.inner.Logs(owner, name, build, stage, step)
}

func (c *droneClient) ServerURL() string {
	return c.server
}
