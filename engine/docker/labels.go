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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/docker/docker/api/types"
	filters2 "github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

func genPathHash(path string) string {
	input, _ := filepath.Abs(path)
	h := sha1.New()
	h.Write([]byte(input))
	bs := h.Sum(nil)
	return hex.EncodeToString(bs)
}

func findContainersWithLabels(cli *client.Client, ctx context.Context, labels map[string]string) ([]string, error) {
	filters := filters2.NewArgs()
	for key, value := range labels {
		filters.Add("label", fmt.Sprintf("%v=%v", key, value))
	}
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: filters})
	if err != nil {
		return nil, err
	}
	logrus.Tracef("containers matching labels: %v", containers)

	var containerIds []string
	for _, container := range containers {
		containerIds = append(containerIds, container.ID)
	}
	return containerIds, nil
}
