package awslambda

import (
	"archive/zip"
	"bytes"
	"fmt"
	"gatehill.io/imposter/remote"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awssession "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/lambda"
	"io"
	"os"
	"path"
)

var defaultIamRoleName = "ImposterLambdaExecutionRole"

func (m Remote) Deploy() (*remote.EndpointDetails, error) {
	if m.config.Url == "" {
		return nil, fmt.Errorf("URL cannot be null")
	}
	//else if token, _ := m.GetObfuscatedToken(); token == "" {
	//	return nil, fmt.Errorf("auth token cannot be null")
	//}

	region, sess := startAwsSession()

	roleArn, err := ensureIamRole(sess, defaultIamRoleName)
	if err != nil {
		logger.Fatal(err)
	}

	// FIXME
	zipContents, err := createDeploymentPackage("3.0.4", m.dir)
	if err != nil {
		logger.Fatal(err)
	}

	svc := lambda.New(sess)

	funcName := m.getFunctionName()
	funcArn, err := ensureFunctionExists(svc, region, funcName, roleArn, zipContents)
	if err != nil {
		return nil, err
	}
	functionUrl, err := ensureUrlConfigured(svc, funcArn)
	if err != nil {
		return nil, err
	}

	details := &remote.EndpointDetails{
		BaseUrl:   functionUrl,
		StatusUrl: remote.MustJoinPath(functionUrl, "/system/status"),

		// UI not supported on lambda
		SpecUrl: remote.MustJoinPath(functionUrl, "/_spec/combined.json"),
	}
	return details, nil
}

func (m Remote) getFunctionName() string {
	return "imposter-example"
}

func startAwsSession() (string, *awssession.Session) {
	region := getAwsRegion()
	sess := awssession.Must(awssession.NewSessionWithOptions(awssession.Options{
		SharedConfigState: awssession.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String(region),
		},
	}))
	return region, sess
}

func ensureFunctionExists(svc *lambda.Lambda, region string, funcName string, roleArn string, zipContents *[]byte) (string, error) {
	var funcArn string
	result, err := checkFunctionExists(svc, funcName)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == lambda.ErrCodeResourceNotFoundException {
				functionArn, err := createFunction(svc, region, funcName, roleArn, zipContents)
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

func createFunction(svc *lambda.Lambda, region string, funcName string, roleArn string, zipContents *[]byte) (arn string, err error) {
	logger.Debugf("creating function: %s in region: %s", funcName, region)

	result, err := svc.CreateFunction(&lambda.CreateFunctionInput{
		Code: &lambda.FunctionCode{
			ZipFile: *zipContents,
		},
		FunctionName: aws.String(funcName),
		Handler:      aws.String("io.gatehill.imposter.awslambda.HandlerV2"),
		MemorySize:   aws.Int64(768),
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

func getAwsRegion() string {
	if defaultRegion, ok := os.LookupEnv("AWS_DEFAULT_REGION"); ok {
		return defaultRegion
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
	assumeRolPolicy := `{
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
		AssumeRolePolicyDocument: &assumeRolPolicy,
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

	logger.Debugf("bundling %d files from workspace", len(files))
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
