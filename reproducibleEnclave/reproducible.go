package reproducibleEnclave

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func GetOracleReproducibleUniqueID() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	log.Println("Running get-enclave-id.sh to figure out the desired unique ID for verification")

	command := exec.Command("/bin/sh", filepath.Join(wd, "get-enclave-id.sh"))
	command.Dir = wd

	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", err
	}

	if err := command.Start(); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(stdout)

	scriptOutput := ""
	lastLine := ""

	for scanner.Scan() {
		lastLine = scanner.Text()
		scriptOutput += lastLine + "\n"
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading from stdout: %w", err)
	}

	if err := command.Wait(); err != nil {
		return "", fmt.Errorf("get-enclave-id.sh failed to complete: %w\nScript output:\n%s", err, scriptOutput)
	}

	expectedUniqueId := lastLine

	if expectedUniqueId == "" || expectedUniqueId == "0000000000000000000000000000000000000000000000000000000000000000" {
		return "", errors.New("couldn't compute expected unique ID of Oracle backend")
	}

	if len(expectedUniqueId) != 64 {
		return "", errors.New("get-enclave-id.sh returned a unique ID of unexpected length")
	}

	return expectedUniqueId, nil
}
