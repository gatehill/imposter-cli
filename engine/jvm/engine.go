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
	provider  *EngineJarProvider
	javaCmd   string
	command   *exec.Cmd
	debouncer debounce.Debouncer
}

func init() {
	engine.RegisterProvider(engine.EngineTypeJvm, func(version string) engine.Provider {
		return GetProvider(version)
	})
	engine.RegisterEngine(engine.EngineTypeJvm, func(configDir string, startOptions engine.StartOptions) engine.MockEngine {
		return BuildEngine(configDir, startOptions)
	})
}

func BuildEngine(configDir string, options engine.StartOptions) engine.MockEngine {
	return &JvmMockEngine{
		configDir: configDir,
		options:   options,
		provider:  GetProvider(options.Version),
		debouncer: debounce.Build(),
	}
}

func (j *JvmMockEngine) Start(wg *sync.WaitGroup) {
	j.startWithOptions(wg, j.options)
}

func (j *JvmMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) {
	if j.javaCmd == "" {
		javaCmd, err := GetJavaCmdPath()
		if err != nil {
			logrus.Fatal(err)
		}
		j.javaCmd = javaCmd
	}
	if !j.provider.Satisfied() {
		if err := j.provider.Provide(engine.PullIfNotPresent); err != nil {
			logrus.Fatal(err)
		}
	}
	args := []string{
		"-jar", j.provider.jarPath,
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

	// watch in case container stops
	go func() {
		j.notifyOnStopBlocking(wg)
	}()
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
	j.notifyOnStopBlocking(wg)
}

func (j *JvmMockEngine) Restart(wg *sync.WaitGroup) {
	wg.Add(1)
	j.Stop(wg)

	// don't pull again
	restartOptions := j.options
	restartOptions.PullPolicy = engine.PullSkip

	j.startWithOptions(wg, restartOptions)
	wg.Done()
}

func (j *JvmMockEngine) notifyOnStopBlocking(wg *sync.WaitGroup) {
	if j.command == nil || j.command.Process == nil {
		logrus.Trace("no subprocess - notifying immediately")
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{})
	}
	pid := strconv.Itoa(j.command.Process.Pid)
	if j.command.ProcessState != nil && j.command.ProcessState.Exited() {
		logrus.Tracef("process with PID: %v already exited - notifying immediately", pid)
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: pid})
	}
	_, err := j.command.Process.Wait()
	if err != nil {
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{
			Id:  pid,
			Err: fmt.Errorf("failed to wait for process with PID: %v: %v", pid, err),
		})
	} else {
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{Id: pid})
	}
}

func (j *JvmMockEngine) StopAllManaged() int {
	processes, err := findImposterJvmProcesses()
	if err != nil {
		logrus.Fatal(err)
	}
	if len(processes) == 0 {
		return 0
	}
	for _, pid := range processes {
		logrus.Tracef("finding JVM process to kill with PID: %d", pid)
		p, err := os.FindProcess(pid)
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Debugf("killing JVM process with PID: %d", pid)
		err = p.Kill()
		if err != nil {
			logrus.Warnf("error killing JVM process with PID: %d: %v", pid, err)
		}
	}
	return len(processes)
}

func (j *JvmMockEngine) GetVersionString() (string, error) {
	if !j.provider.Satisfied() {
		if err := j.provider.Provide(engine.PullIfNotPresent); err != nil {
			return "", err
		}
	}

	output := new(strings.Builder)
	errOutput := new(strings.Builder)

	if j.javaCmd == "" {
		javaCmd, err := GetJavaCmdPath()
		if err != nil {
			return "", err
		}
		j.javaCmd = javaCmd
	}
	args := []string{
		"-jar", j.provider.jarPath,
		"--version",
	}
	command := exec.Command(j.javaCmd, args...)
	command.Stdout = output
	command.Stderr = errOutput
	err := command.Run()

	if err != nil {
		return "", fmt.Errorf("error starting mock engine process: %v\n%v\n%v", err, output, errOutput)
	}
	return engine.SanitiseVersionOutput(output.String()), nil
}
