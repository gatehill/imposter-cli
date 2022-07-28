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
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/plugin"
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"
	"sync"
)

var logger = logging.GetLogger()

func (j *JvmMockEngine) Start(wg *sync.WaitGroup) bool {
	return j.startWithOptions(wg, j.options)
}

func (j *JvmMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) (success bool) {
	if len(options.DirMounts) > 0 {
		logger.Warnf("JVM engine does not support directory mounts - these will be ignored")
	}

	args := []string{
		"--configDir=" + j.configDir,
		fmt.Sprintf("--listenPort=%d", options.Port),
	}
	env := buildEnv(options)
	command := (*j.provider).GetStartCommand(args, env)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	err := command.Start()
	if err != nil {
		logger.Fatalf("failed to exec: %v %v: %v", command.Path, command.Args, err)
	}
	j.debouncer.Register(wg, strconv.Itoa(command.Process.Pid))
	logger.Trace("starting JVM mock engine")
	j.command = command

	up := engine.WaitUntilUp(options.Port, j.shutDownC)

	// watch in case container stops
	go func() {
		j.notifyOnStopBlocking(wg)
	}()

	return up
}

func buildEnv(options engine.StartOptions) []string {
	env := engine.BuildEnv(options, true)
	if options.EnablePlugins {
		logger.Tracef("plugins are enabled")
		pluginDir, err := plugin.EnsurePluginDir(options.Version)
		if err != nil {
			logger.Fatal(err)
		}
		env = append(env, "IMPOSTER_PLUGIN_DIR="+pluginDir)
	}
	if options.EnableFileCache {
		logger.Tracef("file cache enabled")
		fileCacheDir, err := engine.EnsureFileCacheDir()
		if err != nil {
			logger.Fatal(err)
		}
		env = append(env, "IMPOSTER_CACHE_DIR="+fileCacheDir, "IMPOSTER_OPENAPI_REMOTE_FILE_CACHE=true")
	}
	logger.Tracef("engine environment: %v", env)
	return env
}

func (j *JvmMockEngine) StopImmediately(wg *sync.WaitGroup) {
	go func() { j.shutDownC <- true }()
	j.Stop(wg)
}

func (j *JvmMockEngine) Stop(wg *sync.WaitGroup) {
	if j.command == nil {
		logger.Tracef("no process to remove")
		wg.Done()
		return
	}
	if logger.IsLevelEnabled(logrus.TraceLevel) {
		logger.Tracef("stopping mock engine with PID: %v", j.command.Process.Pid)
	} else {
		logger.Info("stopping mock engine")
	}

	err := j.command.Process.Kill()
	if err != nil {
		logger.Fatalf("error stopping engine with PID: %d: %v", j.command.Process.Pid, err)
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
		logger.Trace("no subprocess - notifying immediately")
		j.debouncer.Notify(wg, debounce.AtMostOnceEvent{})
	}
	pid := strconv.Itoa(j.command.Process.Pid)
	if j.command.ProcessState != nil && j.command.ProcessState.Exited() {
		logger.Tracef("process with PID: %v already exited - notifying immediately", pid)
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
		logger.Fatal(err)
	}
	if len(processes) == 0 {
		return 0
	}
	for _, pid := range processes {
		logger.Tracef("finding JVM process to kill with PID: %d", pid)
		p, err := os.FindProcess(pid)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Debugf("killing JVM process with PID: %d", pid)
		err = p.Kill()
		if err != nil {
			logger.Warnf("error killing JVM process with PID: %d: %v", pid, err)
		}
	}
	return len(processes)
}

func (j *JvmMockEngine) GetVersionString() (string, error) {
	if !(*j.provider).Satisfied() {
		if err := (*j.provider).Provide(engine.PullIfNotPresent); err != nil {
			return "", err
		}
	}

	output := new(strings.Builder)
	errOutput := new(strings.Builder)

	args := []string{
		"--version",
	}
	command := (*j.provider).GetStartCommand(args, engine.BuildEnv(j.options, true))
	command.Stdout = output
	command.Stderr = errOutput
	err := command.Run()

	if err != nil {
		return "", fmt.Errorf("error starting mock engine process: %v\n%v\n%v", err, output, errOutput)
	}
	return engine.SanitiseVersionOutput(output.String()), nil
}
