package jvm

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"regexp"
)

func findImposterJvmProcesses() ([]int, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("error listing processes: %v", err)
	}

	var imposterPids []int
	for _, p := range processes {
		cmdline, err := p.CmdlineSlice()
		if err != nil {
			return nil, err
		}
		procName, err := p.Name()
		if err != nil {
			return nil, err
		}
		isImposter, err := isImposterProc(cmdline, procName)
		if err != nil {
			return nil, err
		}
		if !isImposter {
			continue
		}
		logger.Tracef("found JVM Imposter process %d: %v", p.Pid, cmdline)
		imposterPids = append(imposterPids, int(p.Pid))
	}
	return imposterPids, nil
}

func isImposterProc(cmdline []string, procName string) (bool, error) {
	if procName != "java" {
		return false, nil
	}
	for _, arg := range cmdline {
		if matched, _ := regexp.MatchString("/imposter.*\\.jar", arg); matched {
			return true, nil
		}
	}
	return false, nil
}
