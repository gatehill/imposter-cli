package docker

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"strings"
)

type DockerEngineLibrary struct{}

func getLibrary() *DockerEngineLibrary {
	return &DockerEngineLibrary{}
}

func (DockerEngineLibrary) CheckPrereqs() (bool, []string) {
	var msgs []string
	ctx, cli, err := BuildCliClient()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to build Docker client: %v", err))
		return false, msgs
	}

	version, err := cli.ServerVersion(ctx)
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			msgs = append(msgs, fmt.Sprintf("❌ Failed to connect to Docker: %v", err))
			return false, msgs
		} else {
			msgs = append(msgs, fmt.Sprintf("❌ Failed to get Docker version: %v", err))
			return false, msgs
		}
	}
	msgs = append(msgs, "✅ Connected to Docker", fmt.Sprintf("✅ Docker version installed: %v", version.Version))

	return true, msgs
}

func (DockerEngineLibrary) List() ([]engine.EngineMetadata, error) {
	ctx, cli, err := BuildCliClient()
	if err != nil {
		return nil, fmt.Errorf("error building CLI client: %s", err)
	}
	var available []engine.EngineMetadata
	imageSummaries, err := cli.ImageList(ctx, types.ImageListOptions{
		Filters: filters.NewArgs(filters.Arg("reference", engineDockerImage+":*")),
	})
	if err != nil {
		return nil, fmt.Errorf("error listing images: %s", err)
	}
	for _, imageSummary := range imageSummaries {
		for _, tag := range imageSummary.RepoTags {
			available = append(available, engine.EngineMetadata{
				EngineType: engine.EngineTypeDocker,
				Version:    strings.Split(tag, ":")[1],
			})
		}
	}
	return available, nil
}

func (l DockerEngineLibrary) GetProvider(version string) engine.Provider {
	return getProvider(version)
}
