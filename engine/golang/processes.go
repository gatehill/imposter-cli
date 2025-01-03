package golang

import (
	"strconv"

	"gatehill.io/imposter/engine/procutil"
)

var matcher = procutil.ProcessMatcher{
	ProcessName:    "imposter-go",
	CommandPattern: "imposter-go$",
	GetPort: func(cmdline []string) int {
		// Check environment variables
		for _, arg := range cmdline {
			if portStr := procutil.ReadArg([]string{arg}, "IMPOSTER_PORT", ""); portStr != "" {
				if port, err := strconv.Atoi(portStr); err == nil {
					return port
				}
			}
		}
		return 0
	},
}
