package notation

import (
	"bytes"
	"context"
	"fmt"
	"github.com/notaryproject/notation-go"
	_ "github.com/notaryproject/notation-go"
	"github.com/notaryproject/notation-go/dir"
	"github.com/notaryproject/notation-go/registry"
	"github.com/notaryproject/notation-go/verifier"
	_ "github.com/notaryproject/notation-go/verifier"
	"github.com/notaryproject/notation-go/verifier/trustpolicy"
	_ "github.com/notaryproject/notation-go/verifier/trustpolicy"
	"github.com/notaryproject/notation-go/verifier/truststore"
	_ "github.com/notaryproject/notation-go/verifier/truststore"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/model"
	"oras.land/oras-go/v2/registry/remote"
	"os"
	"os/exec"
)

const (
	ValidationFailed = "notation validation failed"
	ValidationPassed = "notation validation passed"
)

//var lock = &sync.Mutex{}

// TrustStore builds the notation trust store
func TrustStore() (string, error) {
	nc := Command{}
	nc.Args = []string{"certificate", "add", "--type", "signingAuthority",
		"--store", model.ServerConfig.Notation.TrustStore, model.ServerConfig.Notation.RootCert}
	nc.Execute()

	if nc.Error != nil {
		return nc.Err, nc.Error
	}

	return nc.Out, nil
}

type Command struct {
	Args    []string
	Subject string
	Out     string
	Err     string
	Error   error
}

// Execute executes notation binary commands
func (nc *Command) Execute() {
	//lock.Lock()
	//defer lock.Unlock()

	var stderr, stdout bytes.Buffer
	cmd := exec.Command(model.ServerConfig.Notation.BinaryDst, nc.Args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", model.ServerConfig.Notation.XdgHomeVar,
		model.ServerConfig.Notation.XdgHomeVal))
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()

	log.Log.Debugf("Notation Command Args: %v", cmd.Args)

	nc.Out = stdout.String()
	nc.Err = stderr.String()
	nc.Error = err
	return
}

func (nc *Command) Verify() {
	tpd := trustPolicyDoc()

	imageVerifier, err := verifier.New(&tpd, truststore.NewX509TrustStore(dir.ConfigFS()), nil)
	if err != nil {
		nc.Err = ValidationFailed
		nc.Error = err
		return
	}

	remoteRepo, err := remote.NewRepository(nc.Subject)
	if err != nil {
		nc.Err = ValidationFailed
		nc.Error = err
		return
	}

	repo := registry.NewRepository(remoteRepo)

	// exampleRemoteVerifyOptions is an example of notation.RemoteVerifyOptions.
	exampleRemoteVerifyOptions := notation.RemoteVerifyOptions{
		ArtifactReference:    nc.Subject,
		PluginConfig:         nil,
		MaxSignatureAttempts: 50,
		UserMetadata:         nil,
	}

	// remote verify core process
	// upon successful verification, the target OCI artifact manifest descriptor
	// and signature verification outcome are returned.
	desc, _, err := notation.Verify(context.Background(), imageVerifier, repo, exampleRemoteVerifyOptions)
	if err != nil {
		nc.Err = ValidationFailed
		nc.Error = err
		return
	}

	nc.Out = ValidationPassed
	log.Log.Debugf("MediaType: %s", desc.MediaType)
	log.Log.Debugf("Digest: %+v", desc.Digest)
	log.Log.Debugf("Size: %d", desc.Size)

	return
}

func trustPolicyDoc() trustpolicy.Document {
	tpd := trustpolicy.Document{
		Version: "1.0",
	}

	tpm := model.TrustPolicy
	var tpa []trustpolicy.TrustPolicy
	for _, p := range tpm.TrustPolicies {
		sv := trustpolicy.SignatureVerification{
			VerificationLevel: p.SignatureVerification.Level,
			Override:          nil,
		}

		if p.SignatureVerification.Override.Revocation != "" {
			va := make(map[trustpolicy.ValidationType]trustpolicy.ValidationAction)
			va[("revocation")] = trustpolicy.ValidationAction(p.SignatureVerification.Override.Revocation)
			sv.Override = va
		}

		tp := trustpolicy.TrustPolicy{
			Name:                  p.Name,
			RegistryScopes:        p.RegistryScopes,
			SignatureVerification: sv,
			TrustStores:           p.TrustStores,
			TrustedIdentities:     p.TrustedIdentities,
		}

		tpa = append(tpa, tp)
	}

	tpd.TrustPolicies = tpa

	return tpd
}
