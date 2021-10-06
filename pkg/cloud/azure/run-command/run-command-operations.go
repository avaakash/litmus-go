package runcommand

import (
	"context"
	"path/filepath"
	"strings"

	experimentTypes "github.com/litmuschaos/litmus-go/pkg/azure/stress-chaos/types"
	azureCommon "github.com/litmuschaos/litmus-go/pkg/cloud/azure/common"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/pkg/errors"
)

func PerformRunCommand(runCommandFuture *experimentTypes.RunCommandFuture, runCommandInput compute.RunCommandInput, subscriptionID, resourceGroup, azureInstanceName, scaleSet string) error {

	if scaleSet == "enable" {
		// Setup and authorize vm client
		vmssClient := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmssClient.Authorizer = authorizer

		scaleSetName, vmId := GetScaleSetNameAndInstanceId(azureInstanceName)
		// Update the VM with the keepAttachedList to detach the specified disks
		future, err := vmssClient.RunCommand(context.TODO(), resourceGroup, scaleSetName, vmId, runCommandInput)
		if err != nil {
			return errors.Errorf("failed to perform run command, err: %v", err)
		}

		runCommandFuture.VmssFuture = future

		return nil
	} else {
		// Setup and authorize vm client
		vmClient := compute.NewVirtualMachinesClient(subscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmClient.Authorizer = authorizer
		future, err := vmClient.RunCommand(context.TODO(), resourceGroup, azureInstanceName, runCommandInput)
		if err != nil {
			return errors.Errorf("failed to perform run command, err: %v", err)
		}

		runCommandFuture.VmFuture = future

		return nil
	}
}

// WaitForRunCommandCompletion waits for the script to complete execution on the instance
func WaitForRunCommandCompletion(runCommandFuture *experimentTypes.RunCommandFuture, subscriptionID, scaleSet string) (compute.RunCommandResult, error) {
	if scaleSet == "enable" {
		// Setup and authorize vm client
		vmssClient := compute.NewVirtualMachineScaleSetVMsClient(subscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmssClient.Authorizer = authorizer

		future := runCommandFuture.VmssFuture
		err = future.WaitForCompletionRef(context.TODO(), vmssClient.Client)
		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("failed to perform run command, err: %v", err)
		}
		result, err := future.Result(vmssClient)
		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("failed to fetch run command results, err: %v", err)
		}

		return result, nil
	} else {
		// Setup and authorize vm client
		vmClient := compute.NewVirtualMachinesClient(subscriptionID)
		authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)

		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("fail to setup authorization, err: %v", err)
		}
		vmClient.Authorizer = authorizer

		future := runCommandFuture.VmFuture
		err = future.WaitForCompletionRef(context.TODO(), vmClient.Client)
		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("failed to perform run command, err: %v", err)
		}

		result, err := future.Result(vmClient)
		if err != nil {
			return compute.RunCommandResult{}, errors.Errorf("failed to fetch run command results, err: %v", err)
		}

		return result, nil
	}
}

// GetScaleSetNameAndInstanceId extracts the scale set name and VM id from the instance name
func GetScaleSetNameAndInstanceId(instanceName string) (string, string) {
	scaleSetAndInstanceId := strings.Split(instanceName, "_")
	return scaleSetAndInstanceId[0], scaleSetAndInstanceId[1]
}

// prepareRunCommandInput prepares the run command with the script, parameters and commandID
func PrepareRunCommandInput(operatingSystem, scriptPath string, parameters *[]compute.RunCommandInputParameter) (compute.RunCommandInput, error) {

	var err error
	var commandId string
	var script []string

	// Setting up command id based on operating system of VM
	if operatingSystem == "windows" {
		commandId = "RunPowerShellScript"
	} else {
		commandId = "RunShellScript"
	}

	// Checking for custom script
	if strings.TrimSpace(scriptPath) != "" {
		scriptPath, _ = filepath.Abs(scriptPath)
	} else {
		return compute.RunCommandInput{}, errors.Errorf("no script provided, either provide a custom script or use default script")
	}

	// Reading script from the file path
	script, err = azureCommon.ReadLines(scriptPath)
	if err != nil {
		return compute.RunCommandInput{}, errors.Errorf("failed to read script, err: %v", err)
	}

	runCommandInput := compute.RunCommandInput{
		CommandID:  &commandId,
		Script:     &script,
		Parameters: parameters,
	}
	return runCommandInput, nil
}

// prepareAbortRunCommandInput prepares abort script command
func PrepareAbortRunCommandInput(operatingSystem, scriptPath string, parameters *[]compute.RunCommandInputParameter) (compute.RunCommandInput, error) {

	var err error
	var commandId string
	var script []string

	// Checking if custom script path is provided or using the default if not
	if strings.TrimSpace(scriptPath) != "" {
		scriptPath, _ = filepath.Abs(scriptPath)
	} else {
		return compute.RunCommandInput{}, errors.Errorf("no script provided, either provide a custom script or use default script")
	}

	// Reading script from the file path
	script, err = azureCommon.ReadLines(scriptPath)
	if err != nil {
		return compute.RunCommandInput{}, errors.Errorf("failed to read script, err: %v", err)
	}

	// Setting up command id based on operating system of VM
	if operatingSystem == "windows" {
		commandId = "RunPowerShellScript"
	} else {
		commandId = "RunShellScript"
	}

	runCommandInput := compute.RunCommandInput{
		CommandID:  &commandId,
		Script:     &script,
		Parameters: parameters,
	}
	return runCommandInput, nil
}
