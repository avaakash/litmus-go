package azure

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/azure/run-command/types"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func PerformRunCommand(experimentDetails *experimentTypes.ExperimentDetails, azureInstanceName string) error {

	runCommandInput, err := PrepareRunCommandInput(experimentDetails)
	if err != nil {
		return errors.Errorf("%v", err)
	}

	if experimentDetails.IsScaleSet == "true" {
		// Setup and authorize vm client
		vmssClient := compute.NewVirtualMachineScaleSetVMsClient(experimentDetails.SubscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmssClient.Authorizer = authorizer

		scaleSetName, vmId := GetScaleSetNameAndInstanceId(azureInstanceName)
		// Update the VM with the keepAttachedList to detach the specified disks
		future, err := vmssClient.RunCommand(context.TODO(), experimentDetails.ResourceGroup, scaleSetName, vmId, runCommandInput)
		if err != nil {
			return errors.Errorf("failed to perform run command, err: %v", err)
		}
		err = future.WaitForCompletionRef(context.TODO(), vmssClient.Client)
		if err != nil {
			return errors.Errorf("failed to perform run command, err: %v", err)
		}
		result, err := future.Result(vmssClient)
		if err != nil {
			return errors.Errorf("failed to fetch run command results, err: %v", err)
		}
		for _, output := range *result.Value {
			log.Infof("RunCommand result: %v", *output.Message)
		}

		return nil
	} else {
		// Setup and authorize vm client
		vmClient := compute.NewVirtualMachinesClient(experimentDetails.SubscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmClient.Authorizer = authorizer

		future, err := vmClient.RunCommand(context.TODO(), experimentDetails.ResourceGroup, azureInstanceName, runCommandInput)
		if err != nil {
			return errors.Errorf("cannot detach disk, err: %v", err)
		}

		err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
		if err != nil {
			return errors.Errorf("failed to perform run command, err: %v", err)
		}

		result, err := future.Result(vmClient)
		if err != nil {
			return errors.Errorf("failed to fetch run command results, err: %v", err)
		}

		for _, output := range *result.Value {
			log.Infof("RunCommand result: %v", *output.Message)
		}
		return nil
	}
}

// GetScaleSetNameAndInstanceId extracts the scale set name and VM id from the instance name
func GetScaleSetNameAndInstanceId(instanceName string) (string, string) {
	scaleSetAndInstanceId := strings.Split(instanceName, "_")
	return scaleSetAndInstanceId[0], scaleSetAndInstanceId[1]
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func PrepareRunCommandInput(experimentDetails *experimentTypes.ExperimentDetails) (compute.RunCommandInput, error) {

	var err error
	var commandId string
	var script []string
	var parameters []compute.RunCommandInputParameter

	if parameters, err = prepareInputParameters(experimentDetails); err != nil {
		return compute.RunCommandInput{}, errors.Errorf("failed to setup input parameters, err: %v", err)
	}
	script, err = readLines("/scripts/script.sh")
	if err != nil {
		return compute.RunCommandInput{}, errors.Errorf("failed to read script, err: %v", err)
	}

	if experimentDetails.OperatingSystem == "windows" {
		commandId = "RunPowerShellScript"
	} else {
		commandId = "RunShellScript"
	}

	runCommandInput := compute.RunCommandInput{
		CommandID:  &commandId,
		Script:     &script,
		Parameters: &parameters,
	}
	return runCommandInput, nil

}

//prepareInputParameters will set the required parameters for the given experiment
func prepareInputParameters(experimentDetails *experimentTypes.ExperimentDetails) ([]compute.RunCommandInputParameter, error) {

	parameters := []compute.RunCommandInputParameter{}

	parameterName := []string{"InstallDependency", "Duration", "ExperimentName", "StressArgs", "AdditionalArgs"}
	parameterValues := []string{experimentDetails.InstallDependency, strconv.Itoa(experimentDetails.ChaosDuration) + "s", experimentDetails.ExperimentType, "", ""}

	parameters = append(parameters, compute.RunCommandInputParameter{
		Name:  &parameterName[0],
		Value: &parameterValues[0],
	})
	// Adding chaos duration to parameter list
	parameters = append(parameters, compute.RunCommandInputParameter{
		Name:  &parameterName[1],
		Value: &parameterValues[1],
	})

	// Adding experiment type to parameter list
	parameters = append(parameters, compute.RunCommandInputParameter{
		Name:  &parameterName[2],
		Value: &parameterValues[2],
	})

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
		return nil, errors.Errorf("stressor for experiment type: %v is not suported", experimentDetails.ExperimentName)
	}

	parameters = append(parameters, compute.RunCommandInputParameter{
		Name:  &parameterName[3],
		Value: &parameterValues[3],
	})

	parameters = append(parameters, compute.RunCommandInputParameter{
		Name:  &parameterName[4],
		Value: &parameterValues[4],
	})

	return parameters, nil
}
