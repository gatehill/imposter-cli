package awslambda

import (
	"errors"
	"fmt"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/engine/awslambda"
	"gatehill.io/imposter/remote"
	"gatehill.io/imposter/stringutil"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"os"
	"path"
	"strings"
	"time"
)

const liveAliasName = "live"

func (m LambdaRemote) Deploy() error {
	region, sess, svc, err := m.initAws()
	if err != nil {
		return err
	}

	roleName := stringutil.GetFirstNonEmpty(m.Config[configKeyIamRoleName], defaultIamRoleName)

	roleArn, err := ensureIamRole(sess, roleName)
	if err != nil {
		logger.Fatal(err)
	}

	engineVersion := engine.GetConfiguredVersion(m.Config[configKeyEngineVersion], true)
	zipContents, err := awslambda.CreateDeploymentPackage(engineVersion, m.Dir)
	if err != nil {
		logger.Fatal(err)
	}

	snapStart := stringutil.ToBool(m.Config[configKeySnapStart])
	funcArn, err := ensureFunctionExists(
		svc,
		region,
		m.getFunctionName(),
		roleArn,
		m.getMemorySize(),
		m.getArchitecture(),
		zipContents,
		snapStart,
	)
	if err != nil {
		return err
	}

	var versionId string
	if stringutil.ToBool(m.Config[configKeyPublishVersion]) {
		versionId, err = publishFunctionVersion(svc, funcArn)
		if err != nil {
			return err
		}
	} else {
		versionId = "$LATEST"
	}

	var arnForUrl string

	createAlias := stringutil.ToBool(m.Config[configKeyCreateAlias])
	if createAlias {
		aliasArn, err := createOrUpdateAlias(svc, funcArn, versionId, liveAliasName)
		if err != nil {
			return err
		}
		arnForUrl = aliasArn
	} else {
		arnForUrl = funcArn
	}

	if _, err = m.ensureUrlConfigured(svc, arnForUrl); err != nil {
		return err
	}

	permitAnonAccess := stringutil.ToBool(m.Config[configKeyAnonAccess])
	if err = configureUrlAccess(svc, arnForUrl, permitAnonAccess); err != nil {
		return err
	}
	return nil
}

func ensureSnapStart(svc *lambda.Lambda, funcArn string, snapStart bool) error {
	var desiredConfig string
	if snapStart {
		desiredConfig = lambda.SnapStartApplyOnPublishedVersions
	} else {
		desiredConfig = lambda.SnapStartApplyOnNone
	}

	configuration, err := svc.GetFunctionConfiguration(&lambda.GetFunctionConfigurationInput{FunctionName: aws.String(funcArn)})
	if err != nil {
		return fmt.Errorf("failed to check snapstart configuration for %v: %v", funcArn, err)
	}
	if *configuration.SnapStart.ApplyOn == desiredConfig {
		logger.Tracef("snapstart set to %v for %v", desiredConfig, funcArn)
		return nil
	}

	logger.Tracef("configuring snapstart for %v", funcArn)
	_, err = svc.UpdateFunctionConfiguration(&lambda.UpdateFunctionConfigurationInput{
		FunctionName: aws.String(funcArn),
		SnapStart: &lambda.SnapStart{
			ApplyOn: aws.String(desiredConfig),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to configure snapstart for %v: %v", funcArn, err)
	}
	logger.Tracef("snapstart set to %v for %v", desiredConfig, funcArn)
	return nil
}

func publishFunctionVersion(svc *lambda.Lambda, funcArn string) (versionId string, err error) {
	if err = awaitLastUpdateSuccess(svc, funcArn); err != nil {
		return "", err
	}

	logger.Tracef("publishing version for %v", funcArn)
	version, err := svc.PublishVersion(&lambda.PublishVersionInput{
		FunctionName: aws.String(funcArn),
	})
	if err != nil {
		return "", err
	}
	versionId = *version.Version
	logger.Debugf("published version %v for %v", versionId, funcArn)
	return versionId, nil
}

func awaitLastUpdateSuccess(svc *lambda.Lambda, funcArn string) error {
	const attempts = 120
	for i := 0; i < attempts; i++ {
		configuration, err := svc.GetFunctionConfiguration(&lambda.GetFunctionConfigurationInput{
			FunctionName: aws.String(funcArn),
		})
		if err != nil {
			return err
		}
		lastUpdateStatus := *configuration.LastUpdateStatus
		logger.Tracef("function %v last update status is %v", funcArn, lastUpdateStatus)
		if lastUpdateStatus == lambda.LastUpdateStatusSuccessful {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timed out after %v attempts waiting for function %v update to succeed", attempts, funcArn)
}

func createOrUpdateAlias(svc *lambda.Lambda, funcArn string, versionId string, aliasName string) (aliasArn string, err error) {
	alias, err := svc.GetAlias(&lambda.GetAliasInput{
		FunctionName: aws.String(funcArn),
		Name:         aws.String(aliasName),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				logger.Tracef("creating alias for function %v version %v", funcArn, versionId)
				alias, err = svc.CreateAlias(&lambda.CreateAliasInput{
					FunctionName:    aws.String(funcArn),
					FunctionVersion: aws.String(versionId),
					Name:            aws.String(aliasName),
				})
				if err != nil {
					return "", err
				}
				aliasArn = *alias.AliasArn
				logger.Debugf("created alias %v to version %v", aliasArn, versionId)
				return aliasArn, nil
			} else {
				return "", fmt.Errorf("failed to get alias %v for function %v: %v", aliasName, funcArn, err)
			}
		} else {
			return "", fmt.Errorf("failed to get alias %v for function %v: %v", aliasName, funcArn, err)
		}
	}

	logger.Debugf("updating alias %v for function %v", aliasName, funcArn)
	alias, err = svc.UpdateAlias(&lambda.UpdateAliasInput{
		FunctionName:    aws.String(funcArn),
		FunctionVersion: aws.String(versionId),
		Name:            aws.String(aliasName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to update alias %v for function %v: %v", aliasName, funcArn, err)
	}
	aliasArn = *alias.AliasArn
	logger.Debugf("updated alias %v to version %v", aliasArn, versionId)
	return aliasArn, nil
}

func (m LambdaRemote) Undeploy() error {
	region, _, svc, err := m.initAws()
	if err != nil {
		return err
	}

	funcName := m.getFunctionName()

	var funcArn string
	funcExists, err := checkFunctionExists(svc, funcName)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				logger.Debugf("function %s does not exist in region %s", funcName, region)
				return nil
			} else {
				return fmt.Errorf("failed to check if function %s exists in region %s: %v", funcName, region, err)
			}
		} else {
			return fmt.Errorf("failed to check if function %s exists in region %s: %v", funcName, region, err)
		}
	} else {
		funcArn = *funcExists.Configuration.FunctionArn
		logger.Tracef("function ARN: %s", funcArn)
	}

	err = m.deleteFunction(funcArn, svc)
	if err != nil {
		return err
	}
	return nil
}

func (m LambdaRemote) GetEndpoint() (*remote.EndpointDetails, error) {
	_, _, svc, err := m.initAws()
	if err != nil {
		return nil, err
	}

	var funcArn string
	funcExists, err := checkFunctionExists(svc, m.getFunctionName())
	if err != nil {
		return nil, err
	} else {
		funcArn = *funcExists.Configuration.FunctionArn
		logger.Tracef("function ARN: %s", funcArn)
	}

	var functionUrl string
	getUrlResult, err := m.checkFunctionUrlConfig(svc, funcArn)
	if err != nil {
		return nil, err
	} else {
		functionUrl = *getUrlResult.FunctionUrl
		logger.Tracef("function URL: %s", functionUrl)
	}

	details := &remote.EndpointDetails{
		BaseUrl:   functionUrl,
		StatusUrl: remote.MustJoinPath(functionUrl, "/system/status"),

		// spec not supported on lambda
		SpecUrl: "",
	}
	return details, nil
}

func (m LambdaRemote) initAws() (region string, sess *awssession.Session, svc *lambda.Lambda, err error) {
	if m.Config[configKeyRegion] == "" {
		return "", nil, nil, fmt.Errorf("region cannot be null")
	}
	region, sess = m.startAwsSession()
	svc = lambda.New(sess)
	return region, sess, svc, nil
}

func configureUrlAccess(svc *lambda.Lambda, funcArn string, anonAccess bool) error {
	const statementId = "PermitAnonymousAccessToFunctionUrl"
	if anonAccess {
		if err := createAnonUrlAccessPolicy(svc, funcArn, statementId); err != nil {
			return err
		}
	} else {
		_, err := svc.RemovePermission(&lambda.RemovePermissionInput{
			FunctionName: aws.String(funcArn),
			StatementId:  aws.String(statementId),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
					logger.Debugf("anonymous URL access permission did not exist")
					return nil
				} else {
					return fmt.Errorf("failed to delete anonymous URL access permission: %v", err)
				}
			} else {
				return fmt.Errorf("failed to delete anonymous URL access permission: %v", err)
			}
		}
		logger.Debugf("deleted anonymous URL access permission")
	}
	return nil
}

func createAnonUrlAccessPolicy(svc *lambda.Lambda, funcArn string, statementId string) error {
	_, err := svc.AddPermission(&lambda.AddPermissionInput{
		StatementId:         aws.String(statementId),
		Action:              aws.String("lambda:InvokeFunctionUrl"),
		FunctionName:        aws.String(funcArn),
		FunctionUrlAuthType: aws.String(lambda.FunctionUrlAuthTypeNone),
		Principal:           aws.String("*"),
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceConflictException {
				logger.Debugf("anonymous URL access permission already exists")
				return nil
			} else {
				return fmt.Errorf("failed to add anonymous URL access permission: %v", err)
			}
		} else {
			return fmt.Errorf("failed to add anonymous URL access permission: %v", err)
		}
	}
	logger.Debugf("added anonymous URL access permission")
	return nil
}

func (m LambdaRemote) getFunctionName() string {
	configuredFuncName := m.Config[configKeyFuncName]
	if configuredFuncName != "" {
		return configuredFuncName
	}
	funcName := path.Base(m.Dir)
	if !strings.HasPrefix(strings.ToLower(funcName), "imposter") {
		funcName = "imposter-" + funcName
	}
	if len(funcName) > 64 {
		return funcName[:64]
	} else {
		return funcName
	}
}

func (m LambdaRemote) startAwsSession() (string, *awssession.Session) {
	region := m.getAwsRegion()
	sess := awssession.Must(awssession.NewSessionWithOptions(awssession.Options{
		SharedConfigState: awssession.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	}))
	return region, sess
}

func ensureFunctionExists(
	svc *lambda.Lambda,
	region string,
	funcName string,
	roleArn string,
	memoryMb int64,
	arch LambdaArchitecture,
	zipContents *[]byte,
	snapStart bool,
) (string, error) {
	var funcArn string
	result, err := checkFunctionExists(svc, funcName)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				functionArn, err := createFunction(
					svc,
					region,
					funcName,
					roleArn,
					memoryMb,
					arch,
					zipContents,
					snapStart,
				)
				if err != nil {
					return "", err
				}
				funcArn = functionArn
			} else {
				return "", fmt.Errorf("failed to check if function %s exists in region %s: %v", funcName, region, err)
			}
		} else {
			return "", fmt.Errorf("failed to check if function %s exists in region %s: %v", funcName, region, err)
		}

	} else {
		funcArn = *result.Configuration.FunctionArn

		if err = ensureSnapStart(svc, funcArn, snapStart); err != nil {
			return "", err
		}
		if err = updateFunctionCode(svc, funcArn, zipContents); err != nil {
			return "", err
		}
	}
	return funcArn, nil
}

func checkFunctionExists(svc *lambda.Lambda, functionName string) (*lambda.GetFunctionOutput, error) {
	result, err := svc.GetFunction(&lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})
	return result, err
}

func createFunction(
	svc *lambda.Lambda,
	region string,
	funcName string,
	roleArn string,
	memoryMb int64,
	arch LambdaArchitecture,
	zipContents *[]byte,
	snapStart bool,
) (arn string, err error) {
	logger.Debugf("creating function: %s in region: %s", funcName, region)

	var desiredConfig string
	if snapStart {
		desiredConfig = lambda.SnapStartApplyOnPublishedVersions
	} else {
		desiredConfig = lambda.SnapStartApplyOnNone
	}

	input := &lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: *zipContents,
		},
		FunctionName:  aws.String(funcName),
		Handler:       aws.String("io.gatehill.imposter.awslambda.HandlerV2"),
		MemorySize:    aws.Int64(memoryMb),
		Role:          aws.String(roleArn),
		Runtime:       aws.String("java11"),
		Architectures: []*string{aws.String(string(arch))},
		Environment:   buildEnv(),
	}

	input.SetSnapStart(&lambda.SnapStart{
		ApplyOn: aws.String(desiredConfig),
	})

	result, err := svc.CreateFunction(input)
	if err != nil {
		var errDetail error
		if awsErr, ok := err.(awserr.Error); ok {
			errDetail = errors.New(awsErr.Error())
		} else {
			errDetail = err
		}
		return "", fmt.Errorf("failed to create function %s in region %s: %v", funcName, region, errDetail)
	}
	logger.Infof("created function: %s with arn: %s", funcName, *result.FunctionArn)
	return *result.FunctionArn, nil
}

func updateFunctionCode(svc *lambda.Lambda, funcArn string, zipContents *[]byte) error {
	logger.Debugf("updating function code for: %s", funcArn)
	_, err := svc.UpdateFunctionCode(&lambda.UpdateFunctionCodeInput{
		FunctionName: aws.String(funcArn),
		ZipFile:      *zipContents,
	})
	if err != nil {
		return err
	}
	logger.Infof("updated function code for: %s", funcArn)
	return nil
}

func (m LambdaRemote) ensureUrlConfigured(svc *lambda.Lambda, funcArn string) (string, error) {
	logger.Debugf("configuring URL for function: %s", funcArn)

	var functionUrl string
	getUrlResult, err := m.checkFunctionUrlConfig(svc, funcArn)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				urlConfigOutput, err := svc.CreateFunctionUrlConfig(&lambda.CreateFunctionUrlConfigInput{
					AuthType:     aws.String(lambda.FunctionUrlAuthTypeNone),
					FunctionName: aws.String(funcArn),
				})
				if err != nil {
					return "", fmt.Errorf("failed to create URL for function: %s: %v", funcArn, err)
				}
				functionUrl = *urlConfigOutput.FunctionUrl
				logger.Debugf("configured function URL: %s", functionUrl)

			} else {
				return "", fmt.Errorf("failed to check if URL config exists for function: %s: %v", funcArn, err)
			}
		} else {
			return "", fmt.Errorf("failed to check if URL config exists for function: %s: %v", funcArn, err)
		}
	} else {
		functionUrl = *getUrlResult.FunctionUrl
		logger.Debugf("function URL already configured: %s", functionUrl)
	}
	return functionUrl, nil
}

func (m LambdaRemote) checkFunctionUrlConfig(
	svc *lambda.Lambda,
	funcArn string,
) (*lambda.GetFunctionUrlConfigOutput, error) {

	input := &lambda.GetFunctionUrlConfigInput{
		FunctionName: aws.String(funcArn),
	}
	if m.shouldCreateAlias() {
		input.Qualifier = aws.String(m.getFunctionAlias())
	}
	logger.Tracef("checking function URL config for %v", input)
	getUrlResult, err := svc.GetFunctionUrlConfig(input)
	return getUrlResult, err
}

func (m LambdaRemote) shouldCreateAlias() bool {
	return stringutil.ToBool(m.Config[configKeyCreateAlias])
}

func (m LambdaRemote) getFunctionAlias() string {
	return liveAliasName
}

func (m LambdaRemote) getAwsRegion() string {
	if defaultRegion, ok := os.LookupEnv("AWS_DEFAULT_REGION"); ok {
		return defaultRegion
	} else if configuredRegion := m.Config[configKeyRegion]; configuredRegion != "" {
		return configuredRegion
	}
	panic("no AWS default region set")
}

func buildEnv() *lambda.Environment {
	env := make(map[string]*string)
	env["IMPOSTER_CONFIG_DIR"] = aws.String("/var/task/config")
	env["JAVA_TOOL_OPTIONS"] = aws.String("-XX:+TieredCompilation -XX:TieredStopAtLevel=1")
	return &lambda.Environment{Variables: env}
}

func (m LambdaRemote) deleteFunction(funcArn string, svc *lambda.Lambda) error {
	logger.Tracef("deleting function: %s", funcArn)
	_, err := svc.DeleteFunction(&lambda.DeleteFunctionInput{
		FunctionName: aws.String(funcArn),
	})
	if err != nil {
		return fmt.Errorf("failed to delete function: %s: %v", funcArn, err)
	}
	logger.Debugf("deleted function: %s", funcArn)
	return nil
}
