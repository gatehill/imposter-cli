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
	"gatehill.io/imposter/engine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	dockerOk, dockerMsgs := engine.GetLibrary(engine.EngineTypeDocker).CheckPrereqs()
	jvmOk, jvmMsgs := engine.GetLibrary(engine.EngineTypeJvmSingleJar).CheckPrereqs()

	var summary string
	if dockerOk || jvmOk {
		summary = "🚀 You should be able to run Imposter, as you have support for one or more engines.\nPass '--engine-type docker' or '--engine-type jvm' when running 'imposter up' to select engine type."
	} else {
		summary = "😭 You may not be able to run Imposter, as you do not have support for at least one engine."
	}
	return fmt.Sprintf(reportTemplate, summary, strings.Join(dockerMsgs, "\n"), strings.Join(jvmMsgs, "\n"))
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
