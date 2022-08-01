package jvm

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/shirou/gopsutil/v3/process"
	"regexp"
	"strconv"
	"strings"
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
		mock := engine.ManagedMock{
			ID:   fmt.Sprintf("%d", p.Pid),
			Name: procName,
			Port: determinePort(cmdline),
		}
		mocks = append(mocks, mock)
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

// determinePort parses the command line arguments to the JVM process
// to determine the listen port
func determinePort(cmdline []string) int {
	for i := range cmdline {
		arg := cmdline[i]
		if matched, _ := regexp.MatchString("--listenPort=", arg); matched {
			// combined form: "--listenPort=NUM"
			port, err := strconv.Atoi(strings.TrimPrefix(arg, "--listenPort="))
			if err != nil {
				return 0
			}
			return port

		} else if (arg == "--listenPort" || arg == "-l)") && i < len(cmdline)-1 {
			// separate form: "--listenPort", "NUM"
			port, err := strconv.Atoi(cmdline[i+1])
			if err != nil {
				return 0
			}
			return port
		}
	}
	return 0
}
