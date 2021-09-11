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

package impostermodel

import (
	"fmt"
	"gatehill.io/imposter/fileutil"
	"github.com/sirupsen/logrus"
)

type ScriptEngine string

const (
	ScriptEngineNone       ScriptEngine = "none"
	ScriptEngineGroovy     ScriptEngine = "groovy"
	ScriptEngineJavaScript ScriptEngine = "javascript"
)

// shorthand for ScriptEngineJavaScript
const jsScriptEngineAlias = "js"

func ParseScriptEngine(scriptEngine string) ScriptEngine {
	engine := ScriptEngine(scriptEngine)
	switch engine {
	case ScriptEngineNone, ScriptEngineGroovy, ScriptEngineJavaScript:
		return engine
	case jsScriptEngineAlias:
		return ScriptEngineJavaScript
	case "":
		return ScriptEngineNone
	default:
		panic(fmt.Errorf("unsupported script engine: %v", scriptEngine))
	}
}

func BuildScriptFilePath(specFilePath string, scriptEngine ScriptEngine, forceOverwrite bool) string {
	var scriptFilePath string
	if IsScriptEngineEnabled(scriptEngine) {
		var scriptEngineExt string
		switch scriptEngine {
		case ScriptEngineJavaScript:
			scriptEngineExt = ".js"
			break
		case ScriptEngineGroovy:
			scriptEngineExt = ".groovy"
			break
		default:
			logrus.Fatal("script engine is disabled")
		}
		scriptFilePath = fileutil.GenerateFilePathAdjacentToFile(specFilePath, scriptEngineExt, forceOverwrite)
	}
	return scriptFilePath
}

func IsScriptEngineEnabled(engine ScriptEngine) bool {
	return len(engine) > 0 && engine != ScriptEngineNone
}
