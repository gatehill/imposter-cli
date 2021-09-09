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
package fileutil

import (
	"github.com/radovskyb/watcher"
	"github.com/sirupsen/logrus"
	"time"
)

const checkIntervalMs = 250

// WatchDir observes changes to the given directory
// and notifies on a channel when they occur.
func WatchDir(dir string) (dirUpdated chan bool) {
	dirUpdated = make(chan bool)

	w := watcher.New()

	if err := w.AddRecursive(dir); err != nil {
		logrus.Warnln(err)
	}

	go func() {
		logrus.Debugf("watching for changes to: %v", dir)
		for {
			select {
			case <-w.Event:
				dirUpdated <- true
				break
			case err := <-w.Error:
				logrus.Warnln(err)
			case <-w.Closed:
				return
			}
		}
	}()

	go func() {
		if err := w.Start(time.Millisecond * checkIntervalMs); err != nil {
			logrus.Warnln(err)
		}
	}()

	return dirUpdated
}
