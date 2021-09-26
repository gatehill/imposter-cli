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
	"gatehill.io/imposter/debounce"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"sync"
)

func removeContainers(d *DockerMockEngine, containerIds []string) {
	logrus.Tracef("removing containers: %v", containerIds)
	wg := &sync.WaitGroup{}

	for _, containerId := range containerIds {
		d.debouncer.Register(wg, containerId)
		removeContainer(d, wg, containerId)
	}
	wg.Wait()
}

func removeContainer(d *DockerMockEngine, wg *sync.WaitGroup, containerId string) {
	go func() {
		ctx, cli, err := BuildCliClient()
		if err != nil {
			logrus.Fatal(err)
		}

		// check it exists
		_, err = cli.ContainerInspect(ctx, containerId)
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to find mock engine container %v to remove: %v", containerId, err)
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		err = cli.ContainerRemove(ctx, containerId, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			if !client.IsErrNotFound(err) {
				logrus.Warnf("failed to remove mock engine container %v: %v", containerId, err)
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId, Err: err})
			} else {
				d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
			}
			return
		}

		notifyOnStopBlocking(d, wg, containerId, cli, ctx)
	}()
}

func notifyOnStopBlocking(d *DockerMockEngine, wg *sync.WaitGroup, containerId string, cli *client.Client, ctx context.Context) {
	statusCh, errCh := cli.ContainerWait(ctx, containerId, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil && !client.IsErrNotFound(err) {
			logrus.Warnf("failed to wait for mock engine container to stop: %v", err)
			d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId, Err: err})
		} else {
			d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
		}
		break
	case <-statusCh:
		logrus.Tracef("mock engine container %v stopped", containerId)
		d.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: containerId})
		break
	}
}

func stopContainersMatchingLabels(d *DockerMockEngine, cli *client.Client, ctx context.Context, containerLabels map[string]string) {
	existingContainerIds, err := findContainersWithLabels(cli, ctx, containerLabels)
	if err != nil {
		logrus.Fatalf("error searching for existing containers: %v", err)
	}
	if len(existingContainerIds) == 0 {
		logrus.Tracef("no existing containers found matching labels: %v", containerLabels)
		return
	}

	logrus.Debugf("replacing %d existing container(s) matching port and directory", len(existingContainerIds))
	removeContainers(d, existingContainerIds)
}
