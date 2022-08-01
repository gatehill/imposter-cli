package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/shirou/gopsutil/v3/process"
	"regexp"
)

func findImposterJvmProcesses() ([]engine.ManagedMock, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("error listing processes: %v", err)
	}

	var mocks []engine.ManagedMock
	for _, p := range processes {
		cmdline, err := p.CmdlineSlice()
		if err != nil {
			continue
		}
		procName, err := p.Name()
		if err != nil {
			continue
		}
		if !isImposterProc(cmdline, procName) {
			continue
		}
		logger.Tracef("found JVM Imposter process %d: %v", p.Pid, cmdline)
		mocks = append(mocks, engine.ManagedMock{ID: fmt.Sprintf("%d", p.Pid), Name: procName})
	}
	return mocks, nil
}

func isImposterProc(cmdline []string, procName string) bool {
	if procName != "java" {
		return false
	}
	for _, arg := range cmdline {
		if matched, _ := regexp.MatchString("/imposter.*\\.jar", arg); matched {
			return true
		}
	}
	return false
}
