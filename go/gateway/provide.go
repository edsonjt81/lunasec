package gateway

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"go.uber.org/config"
	"go.uber.org/zap"
	"log"
)

type Gateways struct {
	KV DynamoKvGateway
	SM AwsSecretsManagerGateway
	S3 AwsS3Gateway
}

type GatewayConfig struct {
	S3Region string `yaml:"region"`
	AccessKeyID string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	LocalstackURL string `yaml:"localstack_url"`
}

func NewGatewayConfig(logger *zap.Logger, provider config.Provider) (gatewayConfig GatewayConfig, err error) {
	err = provider.Get("aws_gateway").Populate(&gatewayConfig)
	if err != nil {
		logger.Error("unable to load aws gateway config", zap.Error(err))
		return
	}
	return
}

func NewAwsSession(logger *zap.Logger, provider config.Provider) (sess *session.Session, err error) {
	var (
		creds *credentials.Credentials
		endpointUrl *string
	)

	gatewayConfig, err := NewGatewayConfig(logger, provider)
	if err != nil {
		log.Println(err)
		return
	}

	sharedConfigEnable := session.SharedConfigEnable
	if gatewayConfig.AccessKeyID != "" && gatewayConfig.SecretAccessKey != "" {
		creds = credentials.NewStaticCredentials(gatewayConfig.AccessKeyID, gatewayConfig.SecretAccessKey, "")
		sharedConfigEnable = session.SharedConfigDisable
	}

	if gatewayConfig.LocalstackURL != "" {
		endpointUrl = aws.String(gatewayConfig.LocalstackURL)
	}

	sessionOptions := session.Options{
		SharedConfigState: sharedConfigEnable,
		Config: aws.Config{
			Credentials: creds,
			Region: aws.String(gatewayConfig.S3Region),
			Endpoint: endpointUrl,
			S3ForcePathStyle: aws.Bool(true),
		},
	}

	sess = session.Must(session.NewSessionWithOptions(sessionOptions))
	return
}

func GetAwsGateways(logger *zap.Logger, provider config.Provider) (gateways Gateways) {
	sess, err := NewAwsSession(logger, provider)
	if err != nil {
		panic(err)
	}

	logger.Debug("loading secrets manager AWS gateway...")
	gateways.SM = NewAwsSecretsManagerGateway(logger, provider, sess)

	logger.Debug("loading dynamodb AWS gateway...")
	gateways.KV = NewDynamoKvGateway(logger, provider, sess)

	logger.Debug("loading s3 AWS gateway...")
	gateways.S3, err = NewAwsS3Gateway(logger, provider.Get("aws_gateway"), sess)
	if err != nil {
		logger.Error("unable to create secrets manager", zap.Error(err))
		panic(err)
	}
	return
}
