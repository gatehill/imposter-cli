package golang

import (
	"fmt"
	"os"
	"os/exec"
	"sync"

	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/procutil"
	"gatehill.io/imposter/logging"
)

var logger = logging.GetLogger()

// GolangMockEngine implements the MockEngine interface for the golang implementation
type GolangMockEngine struct {
	configDir string
	options   engine.StartOptions
	provider  *Provider
	cmd       *exec.Cmd
}

// NewGolangMockEngine creates a new instance of the golang mock engine
func NewGolangMockEngine(configDir string, options engine.StartOptions, provider *Provider) *GolangMockEngine {
	return &GolangMockEngine{
		configDir: configDir,
		options:   options,
		provider:  provider,
	}
}

func (g *GolangMockEngine) Start(wg *sync.WaitGroup) bool {
	return g.startWithOptions(wg, g.options)
}

func (g *GolangMockEngine) startWithOptions(wg *sync.WaitGroup, options engine.StartOptions) (success bool) {
	if len(options.DirMounts) > 0 {
		logger.Warnf("golang engine does not support directory mounts - these will be ignored")
	}

	// Set up environment variables
	env := append(os.Environ(), options.Environment...)
	env = append(env,
		fmt.Sprintf("IMPOSTER_PORT=%d", options.Port),
		fmt.Sprintf("IMPOSTER_CONFIG_DIR=%s", g.configDir),
	)
	if options.LogLevel != "" {
		env = append(env, fmt.Sprintf("IMPOSTER_LOG_LEVEL=%s", options.LogLevel))
	}

	command := (*g.provider).GetStartCommand([]string{}, env)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Start(); err != nil {
		logger.Errorf("failed to start golang mock engine: %v", err)
		return false
	}
	g.cmd = command

	wg.Add(1)
	go g.notifyOnStopBlocking(wg)
	return true
}

func (g *GolangMockEngine) Stop(wg *sync.WaitGroup) {
	if g.cmd != nil && g.cmd.Process != nil {
		logger.Debugf("stopping golang mock engine process: %d", g.cmd.Process.Pid)
		if err := g.cmd.Process.Signal(os.Interrupt); err != nil {
			logger.Warnf("error sending interrupt signal to golang mock engine: %v", err)
			g.StopImmediately(wg)
		}
	}
}

func (g *GolangMockEngine) StopImmediately(wg *sync.WaitGroup) {
	if g.cmd != nil && g.cmd.Process != nil {
		logger.Debugf("force stopping golang mock engine process: %d", g.cmd.Process.Pid)
		if err := g.cmd.Process.Kill(); err != nil {
			logger.Warnf("error killing golang mock engine process: %v", err)
		}
	}
}

func (g *GolangMockEngine) Restart(wg *sync.WaitGroup) {
	g.Stop(wg)
	g.Start(wg)
}

func (g *GolangMockEngine) notifyOnStopBlocking(wg *sync.WaitGroup) {
	defer wg.Done()
	if g.cmd == nil {
		return
	}
	if err := g.cmd.Wait(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			logger.Errorf("error waiting for golang mock engine process: %v", err)
		}
	}
}

func (g *GolangMockEngine) ListAllManaged() ([]engine.ManagedMock, error) {
	return procutil.FindImposterProcesses(matcher)
}

func (g *GolangMockEngine) StopAllManaged() int {
	count, err := procutil.StopManagedProcesses(matcher)
	if err != nil {
		logger.Fatal(err)
	}
	return count
}

func (g *GolangMockEngine) GetVersionString() (string, error) {
	return g.options.Version, nil
}
