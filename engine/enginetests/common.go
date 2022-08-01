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

package enginetests

import (
	"fmt"
	"gatehill.io/imposter/engine"
	"github.com/stretchr/testify/require"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
)

type EngineTestFields struct {
	ConfigDir string
	Options   engine.StartOptions
}

type EngineTestScenario struct {
	Name   string
	Fields EngineTestFields
}

func StartStop(t *testing.T, tests []EngineTestScenario, builder func(scenario EngineTestScenario) engine.MockEngine) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockEngine := builder(tt)
			success := mockEngine.Start(wg)
			if !success {
				t.Fatalf("engine did not start successfully")
			}

			defer func() {
				mockEngine.Stop(wg)
				wg.Wait()
			}()

			checkUp(t, tt.Fields.Options.Port)

			url := fmt.Sprintf("http://localhost:%d/example", tt.Fields.Options.Port)
			resp, err := http.Get(url)
			if err != nil {
				t.Fatalf("failed to invoke mock endpoint: %v", err)
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}

			want := "Hello world"
			got := string(body)
			require.Equal(t, want, got, "expected body to match")
		})
	}
}

func Restart(t *testing.T, tests []EngineTestScenario, builder func(scenario EngineTestScenario) engine.MockEngine) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockEngine := builder(tt)
			success := mockEngine.Start(wg)
			if !success {
				t.Fatalf("engine did not start successfully")
			}

			defer func() {
				mockEngine.Stop(wg)
				wg.Wait()
			}()

			checkUp(t, tt.Fields.Options.Port)

			mockEngine.Restart(wg)
			checkUp(t, tt.Fields.Options.Port)
		})
	}
}

func List(t *testing.T, tests []EngineTestScenario, builder func(scenario EngineTestScenario) engine.MockEngine) {
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			wg := &sync.WaitGroup{}
			mockEngine := builder(tt)
			success := mockEngine.Start(wg)
			if !success {
				t.Fatalf("engine did not start successfully")
			}

			defer func() {
				mockEngine.Stop(wg)
				wg.Wait()
			}()

			checkUp(t, tt.Fields.Options.Port)

			mocks, err := mockEngine.ListAllManaged()
			if err != nil {
				t.Fatalf("failed to list mocks: %s", err)
			}
			require.Equal(t, 1, len(mocks), "expected 1 running mock")
		})
	}
}

func GetFreePort() int {
	if addr, err := net.ResolveTCPAddr("tcp", "localhost:0"); err == nil {
		var l *net.TCPListener
		if l, err = net.ListenTCP("tcp", addr); err == nil {
			defer l.Close()
			return l.Addr().(*net.TCPAddr).Port
		}
	}
	panic("could not find a free port")
}

func checkUp(t *testing.T, port int) {
	if err := engine.CheckMockStatus(port); err != nil {
		t.Fatalf("mock engine down on port: %d: %s", port, err)
	} else {
		t.Logf("mock engine up on port: %d", port)
	}
}
