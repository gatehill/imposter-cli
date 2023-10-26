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
		port := determinePort(cmdline)
		if port == 0 {
			if determineTlsEnabled(cmdline) {
				port = 8443
			} else {
				port = 8080
			}
		}
		mock := engine.ManagedMock{
			ID:   fmt.Sprintf("%d", p.Pid),
			Name: procName,
			Port: port,
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
		if matched, _ := regexp.MatchString("([/\\\\]imposter.*\\.jar$|^io.gatehill.imposter.cmd.ImposterLauncher$)", arg); matched {
			return true
		}
	}
	return false
}

// determinePort parses the command line arguments to the JVM process
// to determine the listen port
func determinePort(cmdline []string) int {
	portRaw := readArg(cmdline, "listenPort", "-l")
	if portRaw != "" {
		port, err := strconv.Atoi(portRaw)
		if err != nil {
			return 0
		}
		return port
	}
	return 0
}

// determineTlsEnabled parses the command line arguments to the JVM process
// to determine if TLS is enabled
func determineTlsEnabled(cmdline []string) bool {
	tlsRaw := readArg(cmdline, "tlsEnabled", "-t")
	if tlsRaw != "" {
		tls, err := strconv.ParseBool(tlsRaw)
		if err != nil {
			return false
		}
		return tls
	}
	return false
}

// readArg parses the command line arguments to the JVM process
// to determine the value of the given arg
func readArg(cmdline []string, longArg string, shortArg string) string {
	for i := range cmdline {
		arg := cmdline[i]
		if matched, _ := regexp.MatchString("--"+longArg+"=", arg); matched {
			// combined form: "--longArg=VAL"
			return strings.TrimPrefix(arg, "--"+longArg+"=")

		} else if (arg == "--"+longArg || arg == "-"+shortArg) && i < len(cmdline)-1 {
			// separate form: "--longArg", "VAL"
			return cmdline[i+1]
		}
	}
	return ""
}
