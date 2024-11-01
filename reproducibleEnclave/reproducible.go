package reproducibleEnclave

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type ReproducedMeasurements struct {
	UniqueID string
	PCRs     [3]string
}

func GetOracleReproducibleMeasurements() (*ReproducedMeasurements, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	log.Println("Running get-enclave-id.sh to compute target enclave measurements")

	command := exec.Command("/bin/sh", filepath.Join(wd, "get-enclave-id.sh"))
	command.Dir = wd

	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := command.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)

	scriptOutput := ""
	lastLine := ""

	for scanner.Scan() {
		lastLine = scanner.Text()
		scriptOutput += lastLine + "\n"
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from stdout: %w", err)
	}

	if err := command.Wait(); err != nil {
		return nil, fmt.Errorf("get-enclave-id.sh failed to complete: %w\nScript output:\n%s", err, scriptOutput)
	}

	outputLines := strings.Split(scriptOutput, "\n")

	uniqueIdLabelIdx := slices.IndexFunc(outputLines, func(element string) bool {
		return element == "Oracle SGX unique ID:"
	})

	pcrLabelIdx := slices.IndexFunc(outputLines, func(element string) bool {
		return element == "Oracle Nitro PCR:"
	})

	if uniqueIdLabelIdx == -1 || len(outputLines) <= uniqueIdLabelIdx+1 {
		return nil, errors.New("SGX unique ID is not found in the script output")
	}

	if pcrLabelIdx == -1 || len(outputLines) <= pcrLabelIdx+3 {
		return nil, errors.New("Nitro PCR values are not found in the script output")
	}

	uniqueID := outputLines[uniqueIdLabelIdx+1]

	pcr0 := outputLines[pcrLabelIdx+1]
	pcr1 := outputLines[pcrLabelIdx+2]
	pcr2 := outputLines[pcrLabelIdx+3]

	if uniqueID == "" || uniqueID == "0000000000000000000000000000000000000000000000000000000000000000" {
		return nil, errors.New("couldn't compute expected SGX unique ID of Oracle backend")
	}

	if len(uniqueID) != 64 {
		return nil, errors.New("get-enclave-id.sh returned a SGX unique ID of unexpected length")
	}

	zeroPcr := "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	if pcr0 == "" || pcr0 == zeroPcr || pcr1 == "" || pcr1 == zeroPcr || pcr2 == "" || pcr2 == zeroPcr {
		return nil, errors.New("couldn't compute expected Nitro PCR values of Oracle backend, or the enclave is in debug mode")
	}

	if len(pcr0) != 96 || len(pcr1) != 96 || len(pcr2) != 96 {
		return nil, errors.New("get-enclave-id.sh returned the Nitro PCR values of unexpected length")
	}

	return &ReproducedMeasurements{
		UniqueID: uniqueID,
		PCRs:     [3]string{pcr0, pcr1, pcr2},
	}, nil
}
