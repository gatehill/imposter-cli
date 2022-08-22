package awslambda

import (
	"fmt"
	"gatehill.io/imposter/logging"
	"gatehill.io/imposter/remote"
	"gatehill.io/imposter/workspace"
	"github.com/araddon/dateparse"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/lambda"
	"net/url"
	"strings"
)

const remoteType = "awslambda"
const defaultRegion = "us-east-1"
const configKeyRegion = "region"
const configKeyFuncName = "functionName"

var configKeys = []string{
	configKeyFuncName,
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
		value = strings.TrimSuffix(value, "/")
		if _, err := url.Parse(value); err != nil {
			return fmt.Errorf("failed to parse URL: %s: %s", value, err)
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

func (m LambdaRemote) getFunctionStatus() (status string, lastModified int, err error) {
	_, sess := m.startAwsSession()
	svc := lambda.New(sess)
	functionName := m.getFunctionName()
	result, err := checkFunctionExists(svc, functionName)
	if err == nil {
		lastModified := 0
		if result.Configuration.LastModified != nil {
			logger.Tracef("function configuration: %+v", result.Configuration)
			if parsed, err := dateparse.ParseStrict(*result.Configuration.LastModified); err == nil {
				lastModified = int(parsed.UnixMilli())
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
