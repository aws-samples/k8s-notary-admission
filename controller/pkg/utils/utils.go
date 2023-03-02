package utils

import (
	"bytes"
	"golang.org/x/sys/unix"

	"os"
	"os/exec"
	"strings"

	"net/url"
	log "notary-admission/pkg/logging"
)

// RemoveFile deletes all in path
func RemoveFile(path string) bool {
	e := os.RemoveAll(path)
	if e != nil {
		log.Log.Errorf("could not remove %s: %v", path, e)
		return false
	}

	return true
}

// CreateFile creates and writes file
func CreateFile(path string, bytes []byte) error {
	var i int
	f, e := os.Create(path)
	defer f.Close()
	if e != nil {
		return e
	}

	i, e = f.Write(bytes)
	if e != nil {
		return e
	}
	log.Log.Debugf("wrote %d bytes to %s", i, path)

	return nil
}

// ReadFile read bytes from file at path
func ReadFile(path string) ([]byte, error) {
	var b []byte
	var err error

	_, err = os.Stat(path) //};  errors.Is(err, os.ErrNotExist) {
	if err != nil {
		log.Log.Errorf("File info issues: %v", err)
	}

	// Read file
	b, err = os.ReadFile(path)
	if err != nil {
		return b, err
	}

	return b, nil
}

// Writable check if file/dir is writable
func Writable(path string) bool {
	return unix.Access(path, unix.W_OK) == nil
}

// FileExists checks if file exists
func FileExists(fileName string) bool {
	_, err := os.Stat(fileName)

	// check if error is "file not exists"
	if os.IsNotExist(err) {
		return false
	}
	return true
}

type VerifiedFile struct {
	FileName  string
	FileFound bool
}

type FileVerifier struct {
	VerifiedFiles []VerifiedFile
}

// VerifyFiles bulk verifies files
func VerifyFiles(files []string) *FileVerifier {
	fv := FileVerifier{
		VerifiedFiles: nil,
	}

	var vfiles []VerifiedFile
	for _, f := range files {
		vf := VerifiedFile{
			FileName:  f,
			FileFound: false,
		}

		if FileExists(f) {
			vf.FileFound = true
		}

		vfiles = append(vfiles, vf)
	}

	fv.VerifiedFiles = vfiles
	return &fv
}

// AccountFromRole parses AWS account ID from role ARN
func AccountFromRole(roleArn string) string {
	a := strings.Split(roleArn, ":")
	if len(a) >= 5 {
		return a[4]
	}
	return ""
}

// AccountFromImage parses AWS account ID from registry url
//func AccountFromImage(registry string) string {
//	a := strings.Split(registry, ".")
//	if len(a) >= 6 {
//		return strings.Split(a[0], "//")[1]
//	}
//	return ""
//}

// RegionFromRegistry parses AWS region ID from registry url
func RegionFromRegistry(registry string) string {
	a := strings.Split(registry, ".")
	if len(a) >= 6 {
		return a[3]
	}
	return ""
}

// RegistryFromImage parses ECR registry from image url
func RegistryFromImage(image string) string {
	if strings.Contains(image, "https://") {
		u, err := url.Parse(image)
		if err != nil {
			log.Log.Errorf("could not get regsitry from %s, %+v", image, err)
			return ""
		}
		return u.Host
	}

	return image[:strings.IndexByte(image, '/')]
}

// CreateDirectory creates dir at path
func CreateDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

// CopyFile copies files, the hard way
func CopyFile(src string, dst string) bool {
	b, e := ReadFile(src)

	if e != nil {
		log.Log.Errorf("could not read file: %v", e)
		return false
	}

	e = CreateFile(dst, b)

	if e != nil {
		log.Log.Errorf("could not write file: %v", e)
		return false
	}

	return true
}

// Chmod sets file mode
func Chmod(path string, mode os.FileMode) bool {
	e := os.Chmod(path, mode)
	if e != nil {
		log.Log.Errorf("could not set %s mode on %s: %v", path, mode.String(), e)
		return false
	}
	return true
}

// Tree retrieves directory tree at path using OS tree binary
func Tree(path string) (string, error) {
	var stderr, stdout bytes.Buffer
	cmd := exec.Command("tree", path)
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	err := cmd.Run()
	if err != nil {
		return stderr.String(), err
	}

	return stdout.String(), err
}
