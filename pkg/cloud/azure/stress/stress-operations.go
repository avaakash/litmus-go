package azure

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/azure/stress-chaos/types"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

//PrepareInputParameters will set the required parameters for the stress chaos experiment
func PrepareInputParameters(experimentDetails *experimentTypes.ExperimentDetails) ([]compute.RunCommandInputParameter, error) {

	parameters := []compute.RunCommandInputParameter{}

	parameterName := []string{"InstallDependency", "Duration", "ExperimentName", "StressArgs", "AdditionalArgs"}
	parameterValues := []string{experimentDetails.InstallDependency, strconv.Itoa(experimentDetails.ChaosDuration), experimentDetails.ExperimentType, "", ""}

	// Adding experiment args to parameter list
	switch experimentDetails.ExperimentType {
	case "cpu-hog":

		log.InfoWithValues("[Info]: Details of Stressor:", logrus.Fields{
			"CPU Core": experimentDetails.CPUcores,
			"Timeout":  experimentDetails.ChaosDuration,
		})

		parameterName[3] = "StressArgs"
		parameterValues[3] = "--cpu " + strconv.Itoa(experimentDetails.CPUcores)

	case "memory-hog":

		log.InfoWithValues("[Info]: Details of Stressor:", logrus.Fields{
			"Number of Workers":  experimentDetails.NumberOfWorkers,
			"Memory Consumption": experimentDetails.MemoryConsumption,
			"Timeout":            experimentDetails.ChaosDuration,
		})
		parameterName[3] = "StressArgs"
		parameterValues[3] = "--vm " + strconv.Itoa(experimentDetails.NumberOfWorkers) + " --vm-bytes " + strconv.Itoa(experimentDetails.MemoryConsumption) + "M"
	case "io-stress":
		var hddbytes string
		if experimentDetails.FilesystemUtilizationBytes == 0 {
			if experimentDetails.FilesystemUtilizationPercentage == 0 {
				hddbytes = "10%"
				log.Info("Neither of FilesystemUtilizationPercentage or FilesystemUtilizationBytes provided, proceeding with a default FilesystemUtilizationPercentage value of 10%")
			} else {
				hddbytes = strconv.Itoa(experimentDetails.FilesystemUtilizationPercentage) + "%"
			}
		} else {
			if experimentDetails.FilesystemUtilizationPercentage == 0 {
				hddbytes = strconv.Itoa(experimentDetails.FilesystemUtilizationBytes) + "G"
			} else {
				hddbytes = strconv.Itoa(experimentDetails.FilesystemUtilizationPercentage) + "%"
				log.Warn("Both FsUtilPercentage & FsUtilBytes provided as inputs, using the FsUtilPercentage value to proceed with stress exp")
			}
		}
		log.InfoWithValues("[Info]: Details of Stressor:", logrus.Fields{
			"io":                experimentDetails.NumberOfWorkers,
			"hdd":               experimentDetails.NumberOfWorkers,
			"hdd-bytes":         hddbytes,
			"Timeout":           experimentDetails.ChaosDuration,
			"Volume Mount Path": experimentDetails.VolumeMountPath,
		})
		if experimentDetails.VolumeMountPath == "" {
			parameterName[3] = "StressArgs"
			parameterValues[3] = "--io " + strconv.Itoa(experimentDetails.NumberOfWorkers) + " --hdd " + strconv.Itoa(experimentDetails.NumberOfWorkers) + " --hdd-bytes " + hddbytes
		} else {
			parameterName[3] = "StressArgs"
			parameterValues[3] = "--io " + strconv.Itoa(experimentDetails.NumberOfWorkers) + " --hdd " + strconv.Itoa(experimentDetails.NumberOfWorkers) + " --hdd-bytes " + hddbytes + " --temp-path " + experimentDetails.VolumeMountPath
		}
		if experimentDetails.CPUcores != 0 {
			parameterName[4] = "AdditionalArgs"
			parameterValues[4] = "--cpu " + strconv.Itoa(experimentDetails.CPUcores)
		}

	default:
		return nil, errors.Errorf("stressor for experiment type: %v is not supported", experimentDetails.ExperimentName)
	}

	// Adding " to start and end of strings
	parameterValues[3] = "\"" + parameterValues[3] + "\""
	parameterValues[4] = "\"" + parameterValues[4] + "\""

	// appending values to parameters
	for i := range parameterValues {
		parameters = append(parameters, compute.RunCommandInputParameter{
			Name:  &parameterName[i],
			Value: &parameterValues[i],
		})
	}

	return parameters, nil
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
