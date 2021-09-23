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

package cmd

import (
	"fmt"
	"gatehill.io/imposter/engine/docker"
	"gatehill.io/imposter/engine/jvm"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os/exec"
	"strings"
)

const reportTemplate = `
SUMMARY
%[1]v

DOCKER ENGINE
%[2]v

JVM ENGINE
%[3]v
`

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check prerequisites for running Imposter",
	Long: `Checks prerequisites for running Imposter, including those needed
by the engines.`,
	Run: func(cmd *cobra.Command, args []string) {
		logrus.Debug("running check up...")
		println(checkPrereqs())
	},
}

func checkPrereqs() string {
	dockerOk, dockerMsgs := checkDocker()
	jvmOk, jvmMsgs := checkJvm()

	var summary string
	if dockerOk || jvmOk {
		summary = "🚀 You should be able to run Imposter, as you have support for one or more engines."
	} else {
		summary = "😭 You may not be able to run Imposter, as you do not have support for at least one engine."
	}
	return fmt.Sprintf(reportTemplate, summary, strings.Join(dockerMsgs, "\n"), strings.Join(jvmMsgs, "\n"))
}

func checkDocker() (bool, []string) {
	var msgs []string
	ctx, cli, err := docker.BuildCliClient()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to build Docker client: %v", err))
		return false, msgs
	}

	version, err := cli.ServerVersion(ctx)
	if err != nil {
		if client.IsErrConnectionFailed(err) {
			msgs = append(msgs, fmt.Sprintf("❌ Failed to connect to Docker: %v", err))
			return false, msgs
		} else {
			msgs = append(msgs, fmt.Sprintf("❌ Failed to get Docker version: %v", err))
			return false, msgs
		}
	}
	msgs = append(msgs, "✅ Connected to Docker", fmt.Sprintf("✅ Docker version installed: %v", version.Version))

	return true, msgs
}

func checkJvm() (bool, []string) {
	var msgs []string
	javaCmdPath, err := jvm.GetJavaCmdPath()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to find JVM installation: %v", err))
		return false, msgs
	}
	msgs = append(msgs, fmt.Sprintf("✅ Found JVM installation: %v", javaCmdPath))

	cmd := exec.Command(javaCmdPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		msgs = append(msgs, fmt.Sprintf("❌ Failed to determine java version: %v", err))
		return false, msgs
	}
	msgs = append(msgs, fmt.Sprintf("✅ Java version installed: %v", string(output)))

	return true, msgs
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
