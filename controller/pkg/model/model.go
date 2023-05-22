package model

import (
	"encoding/json"
	"errors"
	"gopkg.in/yaml.v3"
	"notary-admission/pkg/utils"
	"sort"
)

//const (
//	BinaryMode string = "binary"
//)

// Config stores server YAML configuration
type Config struct {
	Name string `yaml:"name"`
	Log  struct {
		Level    string `yaml:"level"`
		Encoding string `yaml:"encoding"`
	} `yaml:"log"`
	Network struct {
		ServerAddress string `yaml:"serverAddress"`
		Ports         struct {
			Http  string `yaml:"http"`
			Https string `yaml:"https"`
		} `yaml:"ports"`
		Endpoints struct {
			Health     string `yaml:"health"`
			Metrics    string `yaml:"metrics"`
			Validation string `yaml:"validation"`
		} `yaml:"endpoints"`
		TLS struct {
			CertFile string `yaml:"crtFile"`
			KeyFile  string `yaml:"keyFile"`
		} `yaml:"tls"`
	} `yaml:"network"`
	Ecr struct {
		CredentialCache struct {
			Enabled              bool     `yaml:"enabled"`
			PreAuthRegistries    []string `yaml:"preAuthRegistries"`
			CacheRefreshInterval int      `yaml:"cacheRefreshInterval"`
			CacheTimeoutInterval int      `yaml:"cacheTimeoutInterval"`
		} `yaml:"credentialCache"`
		IgnoreRegistries []string `yaml:"ignoreRegistries"`
	} `yaml:"ecr"`
	Notation struct {
		Mode           string `yaml:"mode"`
		DebugEnabled   bool   `yaml:"debugEnabled"`
		DebugFlag      string `yaml:"debugFlag"`
		BinaryDir      string `yaml:"binaryDir"`
		BinarySrc      string `yaml:"binarySrc"`
		BinaryDst      string `yaml:"binaryDst"`
		VersionCommand string `yaml:"cmdVersion"`
		LoginCommand   string `yaml:"cmdLogin"`
		VerifyCommand  string `yaml:"cmdVerify"`
		ListCommand    string `yaml:"cmdList"`
		HomeDir        string `yaml:"homeDirectory"`
		TrustPolicy    string `yaml:"trustPolicy"`
		RootCert       string `yaml:"rootCert"`
		TrustStore     string `yaml:"trustStore"`
		XdgHomeVar     string `yaml:"xdgHomeVariable"`
		XdgHomeVal     string `yaml:"xdgHomeValue"`
		PluginDir      string `yaml:"pluginDir"`
		PluginFile     string `yaml:"pluginFile"`
		SignerEndpoint string `yaml:"signerEndpoint"`
		SignerDebug    bool   `yaml:"signerDebug"`
	} `yaml:"notation"`
	Prometheus struct {
		Name  string  `yaml:"name"`
		Start float64 `yaml:"start"`
		Width float64 `yaml:"width"`
		Count int     `yaml:"count"`
	} `yaml:"prometheus"`
	AwsAccountId     string
	AwsRegion        string
	AwsRole          string
	AwsTokenFilePath string
}

// TrustPolicyModel stores JSON trust policy
type TrustPolicyModel struct {
	Version       string `json:"version"`
	TrustPolicies []struct {
		Name                  string   `json:"name"`
		RegistryScopes        []string `json:"registryScopes"`
		SignatureVerification struct {
			Level    string `json:"level"`
			Override struct {
				Expiry     string `json:"expiry,omitempty"`
				Revocation string `json:"revocation,omitempty"`
			} `json:"override,omitempty"`
		} `json:"signatureVerification"`
		TrustStores       []string `json:"trustStores"`
		TrustedIdentities []string `json:"trustedIdentities"`
	} `json:"trustPolicies"`
}

var (
	ServerConfig     Config
	ConfigFile       string
	TrustPolicyFile  string
	TrustPolicy      TrustPolicyModel
	BypassRegistries map[string]string
)

// Yaml marshals config for YAML output
func (c *Config) Yaml() ([]byte, error) {
	out, err := yaml.Marshal(&c)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Json marshals config for JSON output
func (t *TrustPolicyModel) Json() ([]byte, error) {
	out, err := json.Marshal(&t)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// LoadConfig loads config from provided file
func (c *Config) LoadConfig(configFile string) error {
	if !utils.FileExists(configFile) {
		err := errors.New(configFile + " does not exist")
		return err
	}

	bytes, err := utils.ReadFile(configFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(bytes, &c)
	if err != nil {
		return err
	}

	sort.Strings(c.Ecr.IgnoreRegistries)
	BypassRegistries = make(map[string]string)
	for _, s := range c.Ecr.IgnoreRegistries {
		BypassRegistries[s] = s
	}

	return nil
}

// LoadTrustpolicy loads trust policy from provided file
func (t *TrustPolicyModel) LoadTrustpolicy(file string) error {
	if !utils.FileExists(file) {
		err := errors.New(file + " does not exist")
		return err
	}

	bytes, err := utils.ReadFile(file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &t)
	if err != nil {
		return err
	}

	return nil
}
