/*
Copyright © 2021 Pete Cornish <outofcoffee@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package docker

import (
	"context"
	"gatehill.io/imposter/engine"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
)

type EngineImageProvider struct {
	engine.ProviderOptions
	imageAndTag string
}

func GetProvider(version string) *EngineImageProvider {
	return &EngineImageProvider{
		ProviderOptions: engine.ProviderOptions{
			EngineType: engine.EngineTypeDocker,
			Version:    version,
		},
	}
}

func (d *EngineImageProvider) Provide(policy engine.PullPolicy) error {
	ctx, cli, err := BuildCliClient()
	if err != nil {
		logrus.Fatal(err)
	}
	imageAndTag, err := ensureContainerImage(cli, ctx, d.Version, policy)
	if err != nil {
		logrus.Fatal(err)
	}
	d.imageAndTag = imageAndTag
	return nil
}

func (d *EngineImageProvider) Satisfied() bool {
	return d.imageAndTag != ""
}

func (d *EngineImageProvider) GetEngineType() engine.EngineType {
	return d.EngineType
}

func ensureContainerImage(cli *client.Client, ctx context.Context, imageTag string, imagePullPolicy engine.PullPolicy) (imageAndTag string, e error) {
	imageAndTag = engineDockerImage + ":" + imageTag

	if imagePullPolicy == engine.PullSkip {
		return imageAndTag, nil
	}

	if imagePullPolicy == engine.PullIfNotPresent {
		var hasImage = true
		_, _, err := cli.ImageInspectWithRaw(ctx, imageAndTag)
		if err != nil {
			if client.IsErrNotFound(err) {
				hasImage = false
			} else {
				return "", err
			}
		}
		if hasImage {
			logrus.Debugf("engine image '%v' already present", imageTag)
			return imageAndTag, nil
		}
	}

	err := pullImage(cli, ctx, imageTag, imageAndTag)
	return imageAndTag, err
}

func pullImage(cli *client.Client, ctx context.Context, imageTag string, imageAndTag string) error {
	logrus.Infof("pulling '%v' engine image", imageTag)
	reader, err := cli.ImagePull(ctx, "docker.io/"+imageAndTag, types.ImagePullOptions{})
	if err != nil {
		panic(err)
	}

	var pullLogDestination io.Writer
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		pullLogDestination = os.Stdout
	} else {
		pullLogDestination = ioutil.Discard
	}
	_, err = io.Copy(pullLogDestination, reader)
	if err != nil {
		panic(err)
	}
	return err
}
