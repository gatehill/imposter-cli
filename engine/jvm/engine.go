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

package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type JvmMockEngine struct {
	configDir string
	options   engine.StartOptions
	javaCmd   string
	jarPath   string
	command   *exec.Cmd
	stopMutex *sync.Mutex
	stopping  map[string]bool
}

func BuildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	javaCmd := getJavaCmd()
	logrus.Tracef("using java: %v", javaCmd)

	jarPath := findImposterJar(options.Version, options.PullPolicy)
	logrus.Tracef("using imposter at: %v", jarPath)

	return &JvmMockEngine{
		configDir: configDir,
		options:   options,
		javaCmd:   javaCmd,
		jarPath:   jarPath,
		stopMutex: &sync.Mutex{},
		stopping:  make(map[string]bool),
	}
}

func (j *JvmMockEngine) Start() {
	j.startWithOptions(j.options)
}

func (j *JvmMockEngine) startWithOptions(options engine.StartOptions) {
	args := []string{
		"-jar", j.jarPath,
		"--configDir=" + j.configDir,
		fmt.Sprintf("--listenPort=%v", options.Port),
	}
	command := exec.Command(j.javaCmd, args...)
	command.Env = []string{
		"IMPOSTER_LOG_LEVEL=" + strings.ToUpper(options.LogLevel),
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Start()
	if err != nil {
		logrus.Fatalf("failed to exec imposter: %v %v: %v", j.javaCmd, args, err)
	}
	logrus.Info("mock engine started - press ctrl+c to stop")
	j.command = command
}

func (j *JvmMockEngine) Stop() {
	stopCh := make(chan engine.StopEvent)
	j.TriggerRemovalAndNotify(stopCh)
	<-stopCh
}

func (j *JvmMockEngine) Restart(stopCh chan engine.StopEvent) {
	j.TriggerRemovalAndNotify(stopCh)

	// don't pull again
	restartOptions := j.options
	restartOptions.PullPolicy = engine.PullSkip

	j.startWithOptions(restartOptions)
}

func (j *JvmMockEngine) TriggerRemovalAndNotify(stopCh chan engine.StopEvent) {
	if j.command == nil {
		logrus.Tracef("no PID to remove")
		j.popStoppingInstance(stopCh, engine.StopEvent{})
		return
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine with PID %v", j.command.Process.Pid)
	} else {
		logrus.Info("stopping mock engine")
	}
	j.pushStoppingProcess(j.command.Process.Pid)
	err := j.command.Process.Kill()
	if err != nil {
		logrus.Fatalf("error stopping engine with PID %d: %v", j.command.Process.Pid, err)
	}
	j.NotifyOnStop(stopCh)
}

func (j *JvmMockEngine) NotifyOnStop(stopCh chan engine.StopEvent) {
	if j.command == nil || j.command.Process == nil {
		logrus.Trace("no subprocess - notifying immediately")
		j.popStoppingInstance(stopCh, engine.StopEvent{})
	}
	pid := strconv.Itoa(j.command.Process.Pid)
	if j.command.ProcessState != nil && j.command.ProcessState.Exited() {
		logrus.Tracef("process with PID: %v already exited - notifying immediately", pid)
		j.popStoppingInstance(stopCh, engine.StopEvent{Id: pid})
	}
	go func() {
		_, err := j.command.Process.Wait()
		if err != nil {
			j.popStoppingInstance(stopCh, engine.StopEvent{
				Id:  pid,
				Err: fmt.Errorf("failed to wait for process with PID: %v: %v", pid, err),
			})
		} else {
			j.popStoppingInstance(stopCh, engine.StopEvent{Id: pid})
		}
	}()
}

func (j *JvmMockEngine) BlockUntilStopped() {
	stopCh := make(chan engine.StopEvent)
	j.NotifyOnStop(stopCh)
	<-stopCh
}

func (j *JvmMockEngine) pushStoppingProcess(pid int) {
	j.stopMutex.Lock()
	j.stopping[strconv.Itoa(pid)] = true
	j.stopMutex.Unlock()
}

// popStoppingInstance debounces container stop events
func (j *JvmMockEngine) popStoppingInstance(stopCh chan engine.StopEvent, event engine.StopEvent) {
	if j.stopping[event.Id] {
		j.stopMutex.Lock()
		if j.stopping[event.Id] { // double-guard
			delete(j.stopping, event.Id)
		}
		j.stopMutex.Unlock()
		stopCh <- event
	}
}
