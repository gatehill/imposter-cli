package golang

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/library"
	"gatehill.io/imposter/logging"
)

var providerLogger = logging.GetLogger()

const (
	githubOwner = "imposter-project"
	githubRepo  = "imposter-go"
)

var downloadConfig = library.DownloadConfig{
	LatestBaseUrlTemplate:    fmt.Sprintf("https://github.com/%s/%s/releases/latest/download", githubOwner, githubRepo),
	VersionedBaseUrlTemplate: fmt.Sprintf("https://github.com/%s/%s/releases/download/v%%s", githubOwner, githubRepo),
}

// Provider handles downloading and managing the golang binary
type Provider struct {
	version    string
	binDir     string
	binaryPath string
}

// NewProvider creates a new golang provider instance
func NewProvider(version string, binDir string) *Provider {
	return &Provider{
		version: version,
		binDir:  binDir,
	}
}

func (p *Provider) Satisfied() bool {
	return p.binaryPath != "" && fileExists(p.binaryPath)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (p *Provider) Provide(policy engine.PullPolicy) error {
	if policy == engine.PullSkip {
		p.binaryPath = p.getBinaryPath()
		return nil
	}
	if policy == engine.PullIfNotPresent && p.Satisfied() {
		return nil
	}

	// Download the binary for the current platform
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	goos := runtime.GOOS
	if goos == "darwin" {
		goos = "Darwin"
	}
	fileName := fmt.Sprintf("imposter_%s_%s.tar.gz", goos, arch)
	downloadPath := filepath.Join(p.binDir, fileName)

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll(p.binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %v", err)
	}

	// Download the binary
	if err := library.DownloadBinaryWithConfig(downloadConfig, downloadPath, fileName, p.version, ""); err != nil {
		return fmt.Errorf("failed to download binary: %v", err)
	}

	// Extract the binary
	cmd := exec.Command("tar", "xzf", downloadPath, "-C", p.binDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract binary: %v", err)
	}

	// Clean up the downloaded archive
	if err := os.Remove(downloadPath); err != nil {
		providerLogger.Warnf("failed to clean up downloaded archive: %v", err)
	}

	// Store the binary path
	p.binaryPath = p.getBinaryPath()
	if !fileExists(p.binaryPath) {
		return fmt.Errorf("binary not found at expected path: %s", p.binaryPath)
	}

	return nil
}

func (p *Provider) GetEngineType() engine.EngineType {
	return engine.EngineTypeGolang
}

func (p *Provider) Build(configDir string, startOptions engine.StartOptions) engine.MockEngine {
	return NewGolangMockEngine(configDir, startOptions, p)
}

func (p *Provider) Bundle(configDir string, dest string) error {
	// TODO: Implement if required
	return fmt.Errorf("bundling not implemented for golang engine")
}

func (p *Provider) GetStartCommand(args []string, env []string) *exec.Cmd {
	if !p.Satisfied() {
		if err := p.Provide(engine.PullIfNotPresent); err != nil {
			providerLogger.Fatal(err)
		}
	}
	cmd := exec.Command(p.binaryPath, args...)
	cmd.Env = append(os.Environ(), env...)
	return cmd
}

func (p *Provider) getBinaryPath() string {
	return filepath.Join(p.binDir, "imposter")
}
