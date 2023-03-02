package main

import (
	"context"
	"flag"
	"fmt"
	"notary-admission/pkg/admissioncontroller/verifier"
	"notary-admission/pkg/handlers"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/model"
	"notary-admission/pkg/notation"
	"notary-admission/pkg/utils"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	flagConfigFile := "config file path (string)"
	flagTrustPolicyFile := "config file path (string)"
	flag.StringVar(&model.ConfigFile, "file", "", flagConfigFile)
	flag.StringVar(&model.TrustPolicyFile, "trustPolicyFile", "", flagTrustPolicyFile)
	flag.Parse()

	log.Build("", "")
	if log.Start() != nil {
		panic("could not start logging")
	}

	log.Log.Info("Server starting...")

	var tlsCrt, tlsKey, xdgHomeVal string

	// Ingest config
	if model.ConfigFile == "" {
		panic("input config file path not specified")
	}

	e := model.ServerConfig.LoadConfig(model.ConfigFile)
	if e != nil {
		panic(fmt.Errorf("error ingesting config file: %v", e))
	}

	// Reinitialize logging with ingested settings
	log.Build(model.ServerConfig.Log.Level, model.ServerConfig.Log.Encoding)
	if log.Start() != nil {
		panic("could not restart logging")
	}

	log.Log.Debugf("config file (%s) ingested successfully", model.ConfigFile)

	//port = model.ServerConfig.Network.Ports.Https
	tlsKey = model.ServerConfig.Network.TLS.KeyFile
	tlsCrt = model.ServerConfig.Network.TLS.CertFile
	xdgHomeVal = model.ServerConfig.Notation.XdgHomeVal

	// Verify files/dirs exist
	files := []string{tlsKey, tlsCrt, model.ServerConfig.Notation.BinaryDst, xdgHomeVal}
	fv := utils.VerifyFiles(files)

	for _, f := range fv.VerifiedFiles {
		if !f.FileFound {
			panic(fmt.Sprintf("file not found: %s", f.FileName))
		}
	}

	// Test notation version
	nc := notation.Command{
		Args: []string{model.ServerConfig.Notation.VersionCommand},
	}
	nc.Execute()
	if nc.Error != nil {
		panic(fmt.Sprintf("notation version failed: %s, %s, %v", nc.Out, nc.Err, nc.Error))
	}
	log.Log.Debugf("notation version: %s", nc.Out)

	// Verify IRSA ENV
	region := os.Getenv("AWS_REGION")
	roleArn := os.Getenv("AWS_ROLE_ARN")
	tokenFilePath := os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE")
	model.ServerConfig.AwsRegion = region
	model.ServerConfig.AwsRole = roleArn
	model.ServerConfig.AwsAccountId = utils.AccountFromRole(roleArn)
	model.ServerConfig.AwsTokenFilePath = tokenFilePath

	// Get and log config YAML
	b, err := model.ServerConfig.Yaml()
	if err != nil {
		panic(fmt.Sprintf("error reading config: %v", err))
	}
	log.Log.Debugf("Server Config:\n%s", string(b))

	// Get singleton EcrVerifier
	ecrv := verifier.GetEcrv()
	if ecrv.Error != nil {
		panic("ECR auth not initialized")
	}

	if model.ServerConfig.Ecr.CredentialCache.Enabled {
		err = verifier.Ecrv.LoadPreAuthRegistries()
		if err != nil {
			panic("could not load pre-auth registries")
		}
	}

	// Check/upsert XDG_CONFIG_HOME ENV variable
	xdgVar := model.ServerConfig.Notation.XdgHomeVar
	xdgVal := model.ServerConfig.Notation.XdgHomeVal
	osVal := os.Getenv(xdgVar)
	if xdgVal != osVal {
		log.Log.Infof("trying to set %s env var to %s", xdgVar, xdgVal)
		err = os.Setenv(xdgVar, xdgVal)
		if err != nil {
			panic(fmt.Sprintf("%s not set", xdgVar))
		}
	}

	// Setup server
	//server := handlers.NewServer(port)

	// Graceful shutdown, handle signals
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	//go func() {
	//	log.Log.Info("Server checks passed...")
	//
	//	fmt.Println(server.ListenAndServeTLS(tlsCrt, tlsKey))
	//	if err = server.Shutdown(context.Background()); err != nil {
	//		log.Log.Error(err)
	//	}
	//}()

	// Start server
	go func() {
		errs := run(tlsCrt, tlsKey)
		select {
		case err = <-errs:
			panic(fmt.Sprintf("could not start server, %+v", err))
		}
	}()

	<-done
	fmt.Println("server stopping...")

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	fmt.Println("server exited gracefully")
}

// run starts 3 Go routines with a common error channel
func run(tlsCrt string, tlsKey string) chan error {
	errs := make(chan error)

	// Starting HTTP server
	go func() {
		port := model.ServerConfig.Network.Ports.Http
		log.Log.Infof("starting HTTP listener at %s", port)

		server := handlers.NewServer(port)
		if err := server.ListenAndServe(); err != nil {
			errs <- err
		}
	}()

	// Starting HTTPS server
	go func() {
		port := model.ServerConfig.Network.Ports.Https
		log.Log.Infof("starting HTTPS listener at %s", port)
		server := handlers.NewTlsServer(port)
		if err := server.ListenAndServeTLS(tlsCrt, tlsKey); err != nil {
			errs <- err
		}
	}()

	// Start cron job
	go func() {
		for model.ServerConfig.Ecr.CredentialCache.Enabled {
			time.Sleep(time.Duration(model.ServerConfig.Ecr.CredentialCache.CacheRefreshInterval) * time.Second)
			log.Log.Info("Waking up to refresh cached ECR creds")
			if err := verifier.Ecrv.RefreshCredsCache(); err != nil {
				errs <- err
			}
		}
	}()
	return errs
}
