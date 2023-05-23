package notation

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	log "notary-admission/pkg/logging"
	"notary-admission/pkg/model"
)

const (
	ValidationFailed = "notation validation failed"
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
