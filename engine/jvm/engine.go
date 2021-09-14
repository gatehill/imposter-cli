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
	"gatehill.io/imposter/debounce"
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type JvmMockEngine struct {
	configDir string
	options   engine.StartOptions
	javaCmd   string
	jarPath   string
	command   *exec.Cmd
	debouncer debounce.Debouncer
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
		debouncer: debounce.Build(),
	}
}

func (j *JvmMockEngine) Start() {
	j.startWithOptions(j.options)
}

func (j *JvmMockEngine) startWithOptions(options engine.StartOptions) {
	args := []string{
		"-jar", j.jarPath,
		"--configDir=" + j.configDir,
		fmt.Sprintf("--listenPort=%d", options.Port),
	}
	command := exec.Command(j.javaCmd, args...)
	command.Env = []string{
		"IMPOSTER_LOG_LEVEL=" + strings.ToUpper(options.LogLevel),
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Start()
	if err != nil {
		logrus.Fatalf("failed to exec: %v %v: %v", j.javaCmd, args, err)
	}
	logrus.Info("mock engine started - press ctrl+c to stop")
	j.command = command
}

func (j *JvmMockEngine) Stop() {
	stopCh := make(chan debounce.AtMostOnceEvent)
	j.TriggerRemovalAndNotify(stopCh)
	<-stopCh
}

func (j *JvmMockEngine) Restart(stopCh chan debounce.AtMostOnceEvent) {
	j.TriggerRemovalAndNotify(stopCh)

	// don't pull again
	restartOptions := j.options
	restartOptions.PullPolicy = engine.PullSkip

	j.startWithOptions(restartOptions)
}

func (j *JvmMockEngine) TriggerRemovalAndNotify(stopCh chan debounce.AtMostOnceEvent) {
	if j.command == nil {
		logrus.Tracef("no process to remove")
		j.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{})
		return
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine with PID: %v", j.command.Process.Pid)
	} else {
		logrus.Info("stopping mock engine")
	}

	j.debouncer.Register(strconv.Itoa(j.command.Process.Pid))

	err := j.command.Process.Kill()
	if err != nil {
		logrus.Fatalf("error stopping engine with PID: %d: %v", j.command.Process.Pid, err)
	}
	j.NotifyOnStop(stopCh)
}

func (j *JvmMockEngine) NotifyOnStop(stopCh chan debounce.AtMostOnceEvent) {
	if j.command == nil || j.command.Process == nil {
		logrus.Trace("no subprocess - notifying immediately")
		j.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{})
	}
	pid := strconv.Itoa(j.command.Process.Pid)
	if j.command.ProcessState != nil && j.command.ProcessState.Exited() {
		logrus.Tracef("process with PID: %v already exited - notifying immediately", pid)
		j.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: pid})
	}
	go func() {
		_, err := j.command.Process.Wait()
		if err != nil {
			j.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{
				Id:  pid,
				Err: fmt.Errorf("failed to wait for process with PID: %v: %v", pid, err),
			})
		} else {
			j.debouncer.Notify(stopCh, debounce.AtMostOnceEvent{Id: pid})
		}
	}()
}

func (j *JvmMockEngine) BlockUntilStopped() {
	stopCh := make(chan debounce.AtMostOnceEvent)
	j.NotifyOnStop(stopCh)
	<-stopCh
}
