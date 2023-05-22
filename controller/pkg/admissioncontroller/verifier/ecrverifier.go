package verifier

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"notary-admission/pkg/model"
	"sync"

	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/notation"
	"notary-admission/pkg/utils"
	"os"
	"strings"
	"time"
)

const (
	Session         = "IRSA_CREDS_SESSION"
	EcrPattern      = "<ACCOUNT>.dkr.ecr.<REGION>.amazonaws.com"
	MsgVerifyBypass = "image verification bypassed"
)

var lock = &sync.Mutex{}

// EcrAuthToken provides helper functions for ECR auth token data
type EcrAuthToken struct {
	AuthData types.AuthorizationData
}

// Expiry provides the expiration time
func (e EcrAuthToken) Expiry() time.Time {
	return *e.AuthData.ExpiresAt
}

// BasicAuthCreds decodes tokens and provides the credentials in a slice
func (e EcrAuthToken) BasicAuthCreds() ([]string, error) {
	rawDecodedToken, err := base64.StdEncoding.DecodeString(*e.AuthData.AuthorizationToken)
	if err != nil {
		return nil, fmt.Errorf("could not decode ECR auth token: %w", err)
	}

	return strings.Split(string(rawDecodedToken), ":"), nil
}

type EcrVerifier struct {
	Tokens map[string]EcrAuthToken
	Error  error
}

// Ecrv Singleton used to hold single instance
var Ecrv *EcrVerifier

// GetEcrv creates singleton of EcrVerifier
func GetEcrv() *EcrVerifier {
	if Ecrv == nil {
		lock.Lock()
		defer lock.Unlock()
		//if Ecrv == nil {
		Ecrv = &EcrVerifier{}
		Ecrv.Tokens = make(map[string]EcrAuthToken)
		//}
	}

	return Ecrv
}

// LoadPreAuthRegistries loads registries to be pre-authorized, needed for cross region access
func (e *EcrVerifier) LoadPreAuthRegistries() error {
	// Pre-auth registries
	for _, r := range model.ServerConfig.Ecr.CredentialCache.PreAuthRegistries {
		err := Ecrv.getEcrAuthToken(r)
		if err != nil {
			return err
		}
	}

	r := strings.Replace(strings.Replace(EcrPattern, "<ACCOUNT>",
		model.ServerConfig.AwsAccountId, 1), "<REGION>", model.ServerConfig.AwsRegion, 1)

	log.Log.Debugf("Derived registry = %s", r)

	if _, ok := Ecrv.Tokens[r]; !ok {
		// Get ECR token for registry
		err := Ecrv.getEcrAuthToken(r)
		if err != nil {
			return err
		}
	}

	log.Log.Debugf("Mapped ECR auth tokens: %+v", Ecrv.Tokens)

	return nil
}

// RefreshCredsCache refreshes the cached ECR creds
func (e *EcrVerifier) RefreshCredsCache() error {
	for k, v := range Ecrv.Tokens {
		t := time.Now().Add(time.Second * time.Duration(model.ServerConfig.Ecr.CredentialCache.CacheTimeoutInterval))
		if t.After(v.Expiry()) || time.Now().After(v.Expiry()) {
			err := Ecrv.getEcrAuthToken(k)
			if err != nil {
				return err
			}
			log.Log.Debugf("refreshed creds for %s", k)
		}
		log.Log.Debugf("creds for %s expires at %s", k, v.Expiry().Format(time.RFC3339))

	}
	return nil
}

// getEcrAuthToken get ECR auth token from IAM Roles for Service Account (IRSA) config
func (e *EcrVerifier) getEcrAuthToken(registry string) error {
	podName := os.Getenv("POD_NAME")
	podNamespace := os.Getenv("POD_NAMESPACE")
	region := model.ServerConfig.AwsRegion
	roleArn := model.ServerConfig.AwsRole
	tokenFilePath := model.ServerConfig.AwsTokenFilePath
	apiOverrideEndpoint := os.Getenv("AWS_API_OVERRIDE_ENDPOINT")
	apiOverridePartition := os.Getenv("AWS_API_OVERRIDE_PARTITION")
	apiOverrideRegion := os.Getenv("AWS_API_OVERRIDE_REGION")

	// Verify IRSA ENV is present
	if region == "" || roleArn == "" || tokenFilePath == "" {
		return fmt.Errorf("required environment variables not set, AWS_REGION: %s, AWS_ROLE_ARN: %s, AWS_WEB_IDENTITY_TOKEN_FILE: %s", region, roleArn, tokenFilePath)
	}
	log.Log.Debugf("AWS_REGION: %s, AWS_ROLE_ARN: %s, AWS_WEB_IDENTITY_TOKEN_FILE: %s", region, roleArn, tokenFilePath)

	ctx := context.Background()

	// Custom resolver in case custom endpoints are used
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == ecr.ServiceID && region == apiOverrideRegion {
			log.Log.Debug("AWS ECR basic auth using custom endpoint resolver...")
			log.Log.Debugf("AWS ECR basic auth API override endpoint: %s", apiOverrideEndpoint)
			log.Log.Debugf("AWS ECR basic auth API override partition: %s", apiOverridePartition)
			log.Log.Debugf("AWS ECR basic auth API override region: %s", apiOverrideRegion)
			return aws.Endpoint{
				URL:           apiOverrideEndpoint,
				PartitionID:   apiOverridePartition,
				SigningRegion: apiOverrideRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fall back to its default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(ctx, config.WithEndpointResolverWithOptions(resolver),
		config.WithWebIdentityRoleCredentialOptions(func(options *stscreds.WebIdentityRoleOptions) {
			options.RoleSessionName = Session
		}))

	if err != nil {
		log.Log.Errorf("Error getting cfg: %v", err)
		return fmt.Errorf("failed to load default AWS basic auth config: %w", err)
	}

	//log.Log.Debugf("registry=%s", registry)

	cfg.Region = utils.RegionFromRegistry(registry)
	ecrClient := ecr.NewFromConfig(cfg)
	authOutput, err := ecrClient.GetAuthorizationToken(ctx, nil)

	//input := ecr.GetAuthorizationTokenInput{}
	//input.RegistryIds = []string{ecrRegistry}
	//authOutput, err := ecrClient.GetAuthorizationToken(ctx, &input)

	if err != nil {
		log.Log.Errorf("Error getting ECR Auth Token for %s: %v", registry, err)
		return fmt.Errorf("could not retrieve ECR auth token collection: %w", err)
	}

	e.Tokens[registry] = EcrAuthToken{AuthData: authOutput.AuthorizationData[0]}

	log.Log.Debugf("ECR auth enabled with IRSA - %s pod in the %s namespace",
		podName, podNamespace)

	return nil
}

//type Subjects struct {
//	Images []string
//}

type Response struct {
	ErrorMessage string
	Error        error
	Message      string
	Image        string
	ByPassed     bool
	Warning      string
}

type Verification struct {
	Responses []Response
	Message   string
	Error     error
}

// VerifySubjects verifies images (subjects)
func (e *EcrVerifier) VerifySubjects(images []string) Verification {
	ecrv := GetEcrv()
	v := Verification{}

	for _, i := range images {
		response := Response{}
		registry := utils.RegistryFromImage(i)
		if _, ok := model.BypassRegistries[registry]; ok {
			// bypass image signature verification
			log.Log.Infof("image %s verification was bypassed", i)
			response.Image = i
			response.ByPassed = true
			response.Warning = i + " - " + MsgVerifyBypass
			v.Responses = append(v.Responses, response)
			continue
		}

		if _, ok := ecrv.Tokens[registry]; !ok {
			// Get ECR token for registry
			err := ecrv.getEcrAuthToken(registry)
			if err != nil {
				errMsg := fmt.Errorf("could not get ECR token for %s: %w", registry, err)
				log.Log.Error(errMsg)
				v.Error = errMsg
				v.Message = errMsg.Error()
				return v
			}
		}

		//rawDecodedText, err := base64.StdEncoding.DecodeString(*ecrv.Tokens[registry].AuthData.AuthorizationToken)
		creds, err := ecrv.Tokens[registry].BasicAuthCreds()
		if err != nil {
			errMsg := fmt.Errorf("could not decode ECR token: %w", err)
			log.Log.Error(errMsg)
			v.Error = errMsg
			v.Message = errMsg.Error()
			return v
		}

		log.Log.Debugf("Decoded ECR token: %v", creds)

		//creds := strings.Split(string(rawDecodedText), ":")

		nc := notation.Command{}
		args := []string{model.ServerConfig.Notation.VerifyCommand, "-u", creds[0], "-p", creds[1]}

		nc.Subject = i

		//if model.ServerConfig.Notation.Mode == model.BinaryMode {
		args = append(args, i)

		if model.ServerConfig.Notation.DebugEnabled {
			args = append(args, model.ServerConfig.Notation.DebugFlag)
		}

		if model.ServerConfig.Notation.SignerEndpoint != "" {
			args = append(args, "--plugin-config", fmt.Sprintf("signer-endpoint-url=%s",
				model.ServerConfig.Notation.SignerEndpoint))
		}

		if model.ServerConfig.Notation.SignerDebug {
			args = append(args, "--plugin-config", "debug=true")
		}

		nc.Args = args

		nc.Execute()

		response.Image = nc.Subject
		response.ErrorMessage = nc.Err
		response.Error = nc.Error
		v.Responses = append(v.Responses, response)

		if response.Error != nil {
			return v
		}
	}

	return v
}
