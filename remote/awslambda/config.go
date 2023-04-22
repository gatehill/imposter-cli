package awslambda

import (
	"fmt"
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/remote"
	"gatehill.io/imposter/workspace"
	"github.com/araddon/dateparse"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/lambda"
	"strconv"
)

type LambdaArchitecture string

const (
	LambdaArchitectureX86_64 LambdaArchitecture = "x86_64"
	LambdaArchitectureArm64  LambdaArchitecture = "arm64"
)

const remoteType = "awslambda"
const defaultArchitecture = LambdaArchitectureX86_64
const defaultRegion = "us-east-1"
const defaultMemory = 768

const configKeyAnonAccess = "anonAccess"
const configKeyArchitecture = "architecture"
const configKeyEngineVersion = "engineVersion"
const configKeyFuncName = "functionName"
const configKeyIamRoleName = "iamRoleName"
const configKeyMemory = "memory"
const configKeyRegion = "region"

var configKeys = []string{
	configKeyAnonAccess,
	configKeyArchitecture,
	configKeyEngineVersion,
	configKeyFuncName,
	configKeyIamRoleName,
	configKeyMemory,
	configKeyRegion,
}

var logger = logging.GetLogger()

type LambdaRemote struct {
	remote.RemoteMetadata
}

func Register() {
	remote.Register(remoteType, func(dir string, workspace *workspace.Workspace) (remote.Remote, error) {
		return Load(dir, workspace)
	})
}

func Load(dir string, w *workspace.Workspace) (LambdaRemote, error) {
	c, err := remote.LoadConfig(dir, w, func() *map[string]string {
		return &map[string]string{
			configKeyRegion: defaultRegion,
		}
	})
	if err != nil {
		return LambdaRemote{}, err
	}

	r := LambdaRemote{
		remote.RemoteMetadata{
			Workspace: w,
			Dir:       dir,
			Config:    *c,
		},
	}
	return r, nil
}

func (LambdaRemote) GetType() string {
	return remoteType
}

func (LambdaRemote) GetConfigKeys() []string {
	return configKeys
}

func (m LambdaRemote) SetConfigValue(key string, value string) error {
	if err := m.CheckConfigKey(m.GetConfigKeys(), key); err != nil {
		return err
	}

	if key == configKeyRegion {
		regionFound := false
		for _, p := range endpoints.DefaultPartitions() {
			for r := range p.Regions() {
				if value == r {
					regionFound = true
					break
				}
			}
		}
		if !regionFound {
			return fmt.Errorf("invalid region: %s", value)
		}
	}
	m.Config[key] = value
	return m.SaveConfig()
}

func (m LambdaRemote) GetConfig() (*map[string]string, error) {
	return remote.CloneMap(&m.Config), nil
}

func (m LambdaRemote) GetStatus() (*remote.Status, error) {
	functionStatus, lastModified, err := m.getFunctionStatus()
	if err != nil {
		return nil, err
	}
	status := remote.Status{
		Status:       functionStatus,
		LastModified: lastModified,
	}
	return &status, nil
}

func (m LambdaRemote) getFunctionStatus() (status string, lastModified int64, err error) {
	_, sess := m.startAwsSession()
	svc := lambda.New(sess)
	functionName := m.getFunctionName()
	result, err := checkFunctionExists(svc, functionName)
	if err == nil {
		if result.Configuration.LastModified != nil {
			logger.Tracef("function configuration: %+v", result.Configuration)
			if parsed, err := dateparse.ParseStrict(*result.Configuration.LastModified); err == nil {
				lastModified = parsed.UnixMilli()
			}
		}
		return *result.Configuration.State, lastModified, nil
	} else {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				return "not deployed", 0, nil
			}
		}
	}
	return "", 0, err
}

func (m LambdaRemote) getMemorySize() int64 {
	if configuredMem := m.Config[configKeyMemory]; configuredMem != "" {
		mem, err := strconv.Atoi(configuredMem)
		if err != nil {
			panic(fmt.Errorf("failed to get memory configuration value: %v", err))
		}
		return int64(mem)
	}
	return defaultMemory
}

func (m LambdaRemote) getArchitecture() LambdaArchitecture {
	if configuredArch := m.Config[configKeyArchitecture]; configuredArch != "" {
		return LambdaArchitecture(configuredArch)
	}
	return defaultArchitecture
}
