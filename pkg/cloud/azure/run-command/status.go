package azure

import (
	"regexp"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
)

func GetRunCommandResult(result *compute.RunCommandResult) {
	message := *(*result.Value)[0].Message

	stdout := false
	stderr := false

	for _, line := range strings.Split(strings.TrimSuffix(message, "\n"), "\n") {
		if line == "[stdout]" {
			log.Info("[Info]: Run Command Output: \n")
			stderr = false
			stdout = true
			continue
		}
		if line == "[stderr]" {
			log.Info("[Info]: Run Command Errors: \n")
			stdout = false
			stderr = true
			continue
		}

		if stdout && line != "" {
			log.Infof("[stdout]: %v\n", line)
		}
		if stderr && line != "" {
			log.Errorf("[stderr]: %v\n", line)
		}
	}
}

func CheckRunCommandResultError(result *compute.RunCommandResult) error {
	message := strings.Split(strings.TrimSuffix(*(*result.Value)[0].Message, "\n"), "\n")
	i := 0

	for ; i < len(message) && message[i] != "[stderr]"; i++ {
	}

	// errorCodes := make([][]int)
	var errorCode []int
	errorCode = nil

	if message[i+1] != "" {
		exitCodeRegex := regexp.MustCompile("error:")
		for ; i < len(message); i++ {
			// errorCodes = append(errorCodes, exitCodeRegex.FindStringIndex())
			errorCode = exitCodeRegex.FindStringIndex(message[i])
			break
		}
	}

	if errorCode != nil {
		return errors.Errorf("Script failed due to %v", message[errorCode[0]:])
	}

	return nil

}
