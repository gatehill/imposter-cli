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
	"sync"
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

func (j *JvmMockEngine) Start(wg *sync.WaitGroup) {
	j.startWithOptions(wg, j.options)
}

func (j *JvmMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) {
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
	j.debouncer.Register(wg, strconv.Itoa(command.Process.Pid))
	logrus.Info("mock engine started - press ctrl+c to stop")
	j.command = command

	engine.WaitUntilUp(options.Port)
}

func (j *JvmMockEngine) Stop(wg *sync.WaitGroup) {
	if j.command == nil {
		logrus.Tracef("no process to remove")
		wg.Done()
		return
	}
	if logrus.IsLevelEnabled(logrus.TraceLevel) {
		logrus.Tracef("stopping mock engine with PID: %v", j.command.Process.Pid)
	} else {
		logrus.Info("stopping mock engine")
	}

	err := j.command.Process.Kill()
	if err != nil {
		logrus.Fatalf("error stopping engine with PID: %d: %v", j.command.Process.Pid, err)
	}
	j.NotifyOnStop(wg)
}

func (j *JvmMockEngine) Restart(wg *sync.WaitGroup) {
	innerWg := &sync.WaitGroup{}
	innerWg.Add(1)

	j.Stop(innerWg)
	innerWg.Wait()

	// don't pull again
	restartOptions := j.options
	restartOptions.PullPolicy = engine.PullSkip

	j.startWithOptions(wg, restartOptions)
	wg.Done()
}

func (j *JvmMockEngine) NotifyOnStop(wg *sync.WaitGroup) {
	if j.command == nil || j.command.Process == nil {
		logrus.Trace("no subprocess - notifying immediately")
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{})
	}
	pid := strconv.Itoa(j.command.Process.Pid)
	if j.command.ProcessState != nil && j.command.ProcessState.Exited() {
		logrus.Tracef("process with PID: %v already exited - notifying immediately", pid)
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: pid})
	}
	go func() {
		_, err := j.command.Process.Wait()
		if err != nil {
			j.debouncer.Notify(wg, debounce.AtMostOnceEvent{
				Id:  pid,
				Err: fmt.Errorf("failed to wait for process with PID: %v: %v", pid, err),
			})
		} else {
			j.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: pid})
		}
	}()
}

func (j *JvmMockEngine) BlockUntilStopped() {
	wg := &sync.WaitGroup{}
	j.NotifyOnStop(wg)
	wg.Wait()
}
