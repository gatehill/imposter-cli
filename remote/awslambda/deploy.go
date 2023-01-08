package awslambda

import (
	"archive/zip"
	"bytes"
	"fmt"
	"gatehill.io/imposter/engine"
	"gatehill.io/imposter/remote"
	"gatehill.io/imposter/stringutil"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"io"
	"os"
	"path"
	"strings"
)

var defaultIamRoleName = "ImposterLambdaExecutionRole"

func (m LambdaRemote) Deploy() (*remote.EndpointDetails, error) {
	if m.Config[configKeyRegion] == "" {
		return nil, fmt.Errorf("region cannot be null")
	}

	region, sess := m.startAwsSession()

	roleName := stringutil.GetFirstNonEmpty(m.Config[configKeyIamRoleName], defaultIamRoleName)

	roleArn, err := ensureIamRole(sess, roleName)
	if err != nil {
		logger.Fatal(err)
	}

	engineVersion := engine.GetConfiguredVersion(m.Config[configKeyEngineVersion], true)
	zipContents, err := createDeploymentPackage(engineVersion, m.Dir)
	if err != nil {
		logger.Fatal(err)
	}

	svc := lambda.New(sess)

	funcName := m.getFunctionName()
	funcArn, err := ensureFunctionExists(svc, region, funcName, roleArn, m.getMemorySize(), zipContents)
	if err != nil {
		return nil, err
	}
	functionUrl, err := ensureUrlConfigured(svc, funcArn)
	if err != nil {
		return nil, err
	}

	permitAnonAccess := m.Config[configKeyAnonAccess] == "true"
	err = configureUrlAccess(svc, funcArn, permitAnonAccess)
	if err != nil {
		return nil, err
	}

	details := &remote.EndpointDetails{
		BaseUrl:   functionUrl,
		StatusUrl: remote.MustJoinPath(functionUrl, "/system/status"),

		// spec not supported on lambda
		SpecUrl: "",
	}
	return details, nil
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
	zipContents *[]byte,
) (string, error) {
	var funcArn string
	result, err := checkFunctionExists(svc, funcName)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				functionArn, err := createFunction(svc, region, funcName, roleArn, memoryMb, zipContents)
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
	zipContents *[]byte,
) (arn string, err error) {
	logger.Debugf("creating function: %s in region: %s", funcName, region)

	result, err := svc.CreateFunction(&lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: *zipContents,
		},
		FunctionName: aws.String(funcName),
		Handler:      aws.String("io.gatehill.imposter.awslambda.HandlerV2"),
		MemorySize:   aws.Int64(memoryMb),
		Role:         aws.String(roleArn),
		Runtime:      aws.String("java11"),
		Environment:  buildEnv(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create function %s in region %s: %v", funcName, region, err)
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

func ensureUrlConfigured(svc *lambda.Lambda, funcArn string) (string, error) {
	logger.Debugf("configuring URL for function: %s", funcArn)

	var functionUrl string
	getUrlResult, err := svc.GetFunctionUrlConfig(&lambda.GetFunctionUrlConfigInput{
		FunctionName: aws.String(funcArn),
	})
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

func ensureIamRole(session *awssession.Session, roleName string) (string, error) {
	svc := iam.New(session)
	getRoleResult, err := svc.GetRole(&iam.GetRoleInput{
		RoleName: &roleName,
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
				roleArn, err := createRole(svc, roleName)
				if err != nil {
					return "", err
				}
				return roleArn, nil

			} else {
				logger.Fatalf("failed to get IAM role: %s: %v", roleName, err)
			}
		} else {
			logger.Fatal(err)
		}
	}
	logger.Debugf("using role: %s", *getRoleResult.Role.Arn)
	return *getRoleResult.Role.Arn, nil
}

func createRole(svc *iam.IAM, roleName string) (string, error) {
	description := "Default IAM role for Imposter Lambda"
	assumeRolePolicy := `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}`
	createRoleOutput, err := svc.CreateRole(&iam.CreateRoleInput{
		Description:              &description,
		RoleName:                 &roleName,
		AssumeRolePolicyDocument: &assumeRolePolicy,
	})
	if err != nil {
		logger.Fatalf("failed to create role: %s: %v", roleName, err)
	}
	roleArn := *createRoleOutput.Role.Arn

	arn := "arn:aws:iam::aws:policy/AWSLambdaExecute"
	getPolicyResult, err := svc.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: &arn,
	})
	if err != nil {
		return "", err
	}
	_, err = svc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: getPolicyResult.Policy.Arn,
		RoleName:  &roleName,
	})
	if err != nil {
		return "", err
	}
	logger.Debugf("created role: %s with arn: %s", roleName, roleArn)
	return roleArn, nil
}

func createDeploymentPackage(version string, dir string) (*[]byte, error) {
	binaryPath, err := checkOrDownloadBinary(version)
	if err != nil {
		return nil, err
	}
	local, err := remote.ListLocal(dir)
	if err != nil {
		return nil, err
	}
	pkg, err := addFilesToZip(binaryPath, local)
	if err != nil {
		return nil, err
	}
	logger.Debugf("created deployment package")
	contents := pkg.Bytes()
	return &contents, nil
}

func addFilesToZip(zipPath string, files []string) (*bytes.Buffer, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source zip: %s: %v", zipPath, err)
	}
	defer zr.Close()

	dst := new(bytes.Buffer)
	zw := zip.NewWriter(dst)
	defer zw.Close()

	// copy existing
	for _, zipItem := range zr.File {
		zipItemReader, err := zipItem.OpenRaw()
		if err != nil {
			return nil, err
		}
		header := zipItem.FileHeader
		targetItem, err := zw.CreateRaw(&header)
		_, err = io.Copy(targetItem, zipItemReader)
	}

	logger.Infof("bundling %d files from workspace", len(files))
	for _, localFile := range files {
		logger.Tracef("bundling %s", localFile)
		f, err := zw.Create(path.Join("config", path.Base(localFile)))
		if err != nil {
			return nil, err
		}
		contents, err := readFile(localFile)
		if err != nil {
			return nil, err
		}
		if _, err = f.Write(*contents); err != nil {
			return nil, err
		}
	}

	return dst, nil
}

func readFile(binaryPath string) (*[]byte, error) {
	file, err := os.Open(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %s: %v", binaryPath, err)
	}
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %s: %v", binaryPath, err)
	}
	return &contents, err
}
