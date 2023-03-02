package main

import (
	"flag"
	"fmt"
	log "notary-admission/pkg/logging"
	"notary-admission/pkg/model"
	"notary-admission/pkg/utils"
	"os"
	"notary-admission/pkg/notation"
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

	log.Log.Info("Init starting...")

	var homeDir, xdgHomeVal string

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

	homeDir = model.ServerConfig.Notation.HomeDir
	xdgHomeVal = model.ServerConfig.Notation.XdgHomeVal

	// Verify files/dirs exist
	files := []string{model.ServerConfig.Notation.BinarySrc,
		model.ServerConfig.Notation.RootCert, xdgHomeVal, "signer/" + model.ServerConfig.Notation.PluginFile}
	fv := utils.VerifyFiles(files)

	for _, f := range fv.VerifiedFiles {
		if !f.FileFound {
			panic(fmt.Sprintf("file not found: %s", f.FileName))
		}
	}

	if !utils.Writable(xdgHomeVal) {
		panic(fmt.Sprintf("%s is not writable", xdgHomeVal))
	}

	// Delete notation home
	d := utils.RemoveFile(homeDir)
	if !d {
		log.Log.Errorf("could not delete %s", homeDir)
	}

	// Create notation home
	err := utils.CreateDirectory(homeDir)
	if err != nil {
		panic(fmt.Sprintf("could not create notation home: %s", homeDir))
	}

	// Check writable dir
	if !utils.Writable(homeDir) {
		panic(fmt.Sprintf("%s is not writable", homeDir))
	}

	// Ingest trust policy
	if model.TrustPolicyFile == "" {
		panic("TrustPolicy file path not specified")
	}
	e = model.TrustPolicy.LoadTrustpolicy(model.TrustPolicyFile)
	if e != nil {
		panic(fmt.Errorf("error ingesting trust policy file: %v", e))
	}

	// Get and log config YAML
	b, err := model.ServerConfig.Yaml()
	if err != nil {
		panic(fmt.Sprintf("error reading config: %v", err))
	}
	log.Log.Debugf("Server Config:\n%s", string(b))

	// Get and log TP JSON
	b, err = model.TrustPolicy.Json()
	if err != nil {
		panic(fmt.Sprintf("error reading trust policy: %v", err))
	}
	log.Log.Debugf("Trust policy:\n%s", string(b))

	// Write trust policy
	trustPolicyPath := homeDir + "/" + model.ServerConfig.Notation.TrustPolicy
	err = utils.CreateFile(trustPolicyPath, b)
	if err != nil {
		panic(fmt.Sprintf("error writing trust policy: %v", err))
	}

	var out string
	// Tree config dir
	out, err = utils.Tree(model.ServerConfig.Notation.XdgHomeVal)
	if err != nil {
		log.Log.Errorf("tree of %s failed: %v", model.ServerConfig.Notation.XdgHomeVal, err)
	}
	log.Log.Debugf("tree of %s:\n%s", model.ServerConfig.Notation.XdgHomeVal, out)

	// Read TP
	b, err = utils.ReadFile(trustPolicyPath)
	if err != nil {
		log.Log.Errorf("could not read file: %s, %v", model.ServerConfig.Notation.XdgHomeVal, err)
	}
	log.Log.Debugf("read trust policy: %s", string(b))

	// Create notation bin dir
	binaryDir := model.ServerConfig.Notation.BinaryDir
	err = utils.CreateDirectory(binaryDir)
	if err != nil {
		panic(fmt.Sprintf("could not create binary dir: %s", binaryDir))
	}

	// Check writable
	if !utils.Writable(binaryDir) {
		panic(fmt.Sprintf("%s is not writable", binaryDir))
	}

	// Copy notation binary
	binarySrc := model.ServerConfig.Notation.BinarySrc
	binaryPath := binaryDir + "/notation"
	if !utils.CopyFile(binarySrc, binaryPath) {
		panic(fmt.Sprintf("could not copy %s to %s", binarySrc, binaryPath))
	}

	// Chmod signer plugin
	fm := os.FileMode(0755)
	if !utils.Chmod(binaryPath, fm) {
		panic(fmt.Sprintf("could not set %s file mode to %s", binaryPath, fm))
	}

	// Configure trust store
	out, err = notation.TrustStore()
	if err != nil {
		panic(fmt.Sprintf("could not configure notation trust store: %s, %v", out, err))
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

	// Create notation plugin dir
	pluginDir := model.ServerConfig.Notation.PluginDir
	err = utils.CreateDirectory(pluginDir)
	if err != nil {
		panic(fmt.Sprintf("could not create plugin dir: %s", pluginDir))
	}

	// Check writable
	if !utils.Writable(pluginDir) {
		panic(fmt.Sprintf("%s is not writable", pluginDir))
	}

	// Copy signer plugin
	pluginFile := model.ServerConfig.Notation.PluginFile
	pluginPath := pluginDir + "/" + pluginFile
	if !utils.CopyFile("signer/"+pluginFile, pluginPath) {
		panic(fmt.Sprintf("could not copy %s to %s", "signer/"+pluginFile, pluginPath))
	}

	// Chmod signer plugin
	if !utils.Chmod(pluginPath, fm) {
		panic(fmt.Sprintf("could not set %s file mode to %s", pluginPath, fm))
	}

	// Tree config dir
	out, err = utils.Tree(xdgVal)
	if err != nil {
		log.Log.Errorf("tree of %s failed: %v", xdgVal, err)
	}
	log.Log.Debugf("tree of %s:\n%s", xdgVal, out)

	log.Log.Info("Init completed successfully...")
}
