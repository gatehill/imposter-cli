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
	binaryName  = "imposter-go"
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
	binaryPath, err := ensureBinary(p.version, policy, p.binDir)
	if err != nil {
		return err
	}
	p.binaryPath = binaryPath
	return nil
}

func ensureBinary(version string, policy engine.PullPolicy, binDir string) (string, error) {
	return checkOrDownloadBinary(version, policy, binDir)
}

func checkOrDownloadBinary(version string, policy engine.PullPolicy, binDir string) (string, error) {
	// Get the binary path for this version
	binaryPath := filepath.Join(binDir, binaryName)
	if policy == engine.PullSkip {
		return binaryPath, nil
	}

	if policy == engine.PullIfNotPresent {
		if _, err := os.Stat(binaryPath); err != nil {
			if !os.IsNotExist(err) {
				return "", fmt.Errorf("failed to stat: %v: %v", binaryPath, err)
			}
		} else {
			providerLogger.Debugf("engine binary '%v' already present", version)
			providerLogger.Tracef("binary for version %v found at: %v", version, binaryPath)
			return binaryPath, nil
		}
	}

	if err := downloadAndExtractBinary(version, binDir); err != nil {
		return "", fmt.Errorf("failed to fetch binary: %v", err)
	}
	providerLogger.Tracef("using imposter-go at: %v", binaryPath)
	return binaryPath, nil
}

func downloadAndExtractBinary(version string, binDir string) error {
	// Create bin directory if it doesn't exist
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %v", err)
	}

	// Get platform-specific filename
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	}
	goos := runtime.GOOS
	if goos == "darwin" {
		goos = "Darwin"
	}
	fileName := fmt.Sprintf("imposter_%s_%s.tar.gz", goos, arch)
	downloadPath := filepath.Join(binDir, fileName)

	// Download the binary
	if err := library.DownloadBinaryWithConfig(downloadConfig, downloadPath, fileName, version, ""); err != nil {
		return fmt.Errorf("failed to download binary: %v", err)
	}

	// Extract the binary
	cmd := exec.Command("tar", "xzf", downloadPath, "-C", binDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract binary: %v", err)
	}

	// Rename the extracted binary
	oldPath := filepath.Join(binDir, "imposter")
	newPath := filepath.Join(binDir, binaryName)
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename binary: %v", err)
	}

	// Clean up the downloaded archive
	if err := os.Remove(downloadPath); err != nil {
		providerLogger.Warnf("failed to clean up downloaded archive: %v", err)
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
	return filepath.Join(p.binDir, binaryName)
}
