package jvm

import (
	"strconv"

	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/procutil"
)

var matcher = procutil.ProcessMatcher{
	ProcessName:    "java",
	CommandPattern: "([/\\\\]imposter.*\\.jar$|^io.gatehill.imposter.cmd.ImposterLauncher$)",
	GetPort: func(cmdline []string) int {
		portRaw := procutil.ReadArg(cmdline, "listenPort", "l")
		if portRaw != "" {
			if port, err := strconv.Atoi(portRaw); err == nil {
				return port
			}
		}
		if isTlsEnabled(cmdline) {
			return 8443
		}
		return 0
	},
}

func isTlsEnabled(cmdline []string) bool {
	tlsRaw := procutil.ReadArg(cmdline, "tlsEnabled", "t")
	if tlsRaw != "" {
		if tls, err := strconv.ParseBool(tlsRaw); err == nil {
			return tls
		}
	}
	return false
}

func findImposterJvmProcesses() ([]engine.ManagedMock, error) {
	return procutil.FindImposterProcesses(matcher)
}
