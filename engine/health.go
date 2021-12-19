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

package engine

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"time"
)

const defaultStartTimeout = 30 * time.Second

func getStartTimeout() time.Duration {
	startTimeout := viper.GetInt("startTimeout")
	if startTimeout == 0 {
		return defaultStartTimeout
	}
	return time.Duration(startTimeout) * time.Second
}

func WaitUntilUp(port int, shutDownC chan bool) (success bool) {
	url := fmt.Sprintf("http://localhost:%d/system/status", port)
	logrus.Tracef("waiting for mock engine to come up at %v", url)

	startedC := make(chan bool)
	max := time.NewTimer(getStartTimeout())
	defer max.Stop()

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			resp, err := http.Get(url)
			if err != nil {
				continue
			}
			if _, err := io.ReadAll(resp.Body); err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode == 200 {
				startedC <- true
				break
			}
		}
	}()

	finished := false
	select {
	case <-max.C:
		finished = true
		logrus.Fatal("timed out waiting for engine to start")
		return false
	case <-startedC:
		finished = true
		logrus.Tracef("engine started")
		return true
	case <-shutDownC:
		if !finished {
			logrus.Debugf("aborted health probe")
		}
		return false
	}
}
