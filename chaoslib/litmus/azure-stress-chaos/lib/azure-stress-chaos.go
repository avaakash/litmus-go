package lib

import (
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/azure/stress-chaos/types"
	clients "github.com/litmuschaos/litmus-go/pkg/clients"
	runCommand "github.com/litmuschaos/litmus-go/pkg/cloud/azure/run-command"
	azure "github.com/litmuschaos/litmus-go/pkg/cloud/azure/stress"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
)

var (
	err           error
	inject, abort chan os.Signal
)

func PrepareAzureStressChaos(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	// inject channel is used to transmit signal notifications
	inject = make(chan os.Signal, 1)
	// Catch and relay certain signal(s) to inject channel
	signal.Notify(inject, os.Interrupt, syscall.SIGTERM)

	// abort channel is used to transmit signal notifications.
	abort = make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt, syscall.SIGTERM)

	// Waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	//  get the instance name or list of instance names
	instanceNameList := strings.Split(experimentsDetails.AzureInstanceNames, ",")
	if len(instanceNameList) == 0 {
		return errors.Errorf("no instance name found")
	}

	// prepare run command parameters
	parameters, err := azure.PrepareInputParameters(experimentsDetails)
	if err != nil {
		return errors.Errorf("failed to prepare chaos parameters, err: %v", err)
	}

	runCommandInput, err := runCommand.PrepareRunCommandInput(experimentsDetails.OperatingSystem, experimentsDetails.ScriptPath, &parameters)
	if err != nil {
		return errors.Errorf("failed to prepare run command input, err: %v", err)
	}

	abortRunCommandInput, err := runCommand.PrepareAbortRunCommandInput(experimentsDetails.OperatingSystem, experimentsDetails.ScriptPath, &parameters)
	if err != nil {
		return errors.Errorf("failed to prepare abort command input, err: %v", err)
	}
	// watching for the abort signal and revert the chaos
	go abortWatcher(experimentsDetails, instanceNameList, abortRunCommandInput)

	switch strings.ToLower(experimentsDetails.Sequence) {
	case "serial":
		if err = injectChaosInSerialMode(experimentsDetails, instanceNameList, runCommandInput, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	case "parallel":
		if err = injectChaosInParallelMode(experimentsDetails, instanceNameList, runCommandInput, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
			return err
		}
	default:
		return errors.Errorf("%v sequence is not supported", experimentsDetails.Sequence)
	}

	// Waiting for the ramp time after chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}
	return nil
}

func injectChaosInParallelMode(experimentsDetails *experimentTypes.ExperimentDetails, instanceNameList []string, runCommandInput compute.RunCommandInput, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {
	select {
	case <-inject:
		// Stopping the chaos execution, if abort signal received
		os.Exit(0)
	default:
		// ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
		ChaosStartTimeStamp := time.Now()
		duration := int(time.Since(ChaosStartTimeStamp).Seconds())

		for duration < experimentsDetails.ChaosDuration {

			runCommandFutures := []experimentTypes.RunCommandFuture{}

			log.Infof("[Info]: Target instanceName list, %v", instanceNameList)

			if experimentsDetails.EngineName != "" {
				msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on Azure instance"
				types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
				events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
			}

			// Running scripts parallely
			for _, vmName := range instanceNameList {
				log.Infof("[Chaos]: Running script on the Azure instance: %v", vmName)
				runCommandFuture := experimentTypes.RunCommandFuture{}
				if err := runCommand.PerformRunCommand(&runCommandFuture, runCommandInput, experimentsDetails.SubscriptionID, experimentsDetails.ResourceGroup, vmName, experimentsDetails.ScaleSet); err != nil {
					return errors.Errorf("unable to run script on azure instance, err: %v", err)
				}
				runCommandFutures = append(runCommandFutures, runCommandFuture)
			}

			// Run probes during chaos
			if len(resultDetails.ProbeDetails) != 0 {
				if err = probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
					return err
				}
			}

			// Wait for Chaos interval
			log.Infof("[Wait]: Waiting for chaos interval of %vs", experimentsDetails.ChaosInterval)
			common.WaitForDuration(experimentsDetails.ChaosInterval)

			for i, vmName := range instanceNameList {
				log.Infof("[Wait]: Waiting for script execution completion on instance: %v", vmName)
				result, err := runCommand.WaitForRunCommandCompletion(&runCommandFutures[i], experimentsDetails.SubscriptionID, experimentsDetails.ScaleSet)
				if err != nil {
					return errors.Errorf("%v", err)
				}
				runCommand.GetRunCommandResult(&result)
				if err = azure.CheckRunCommandResultError(&result); err != nil {
					return err
				}
			}

			duration = int(time.Since(ChaosStartTimeStamp).Seconds())
		}
	}
	return nil
}

func injectChaosInSerialMode(experimentsDetails *experimentTypes.ExperimentDetails, instanceNameList []string, runCommandInput compute.RunCommandInput, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {
	select {
	case <-inject:
		// Stopping the chaos execution, if abort signal received
		os.Exit(0)
	default:
		// ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
		ChaosStartTimeStamp := time.Now()
		duration := int(time.Since(ChaosStartTimeStamp).Seconds())

		for duration < experimentsDetails.ChaosDuration {

			log.Infof("[Info]: Target instanceName list, %v", instanceNameList)

			if experimentsDetails.EngineName != "" {
				msg := "Injecting " + experimentsDetails.ExperimentName + " chaos on Azure instance"
				types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
				events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
			}

			// Running scripts serially
			for _, vmName := range instanceNameList {

				log.Infof("[Chaos]: Running script on the Azure instance: %v", vmName)
				runCommandFuture := experimentTypes.RunCommandFuture{}
				if err := runCommand.PerformRunCommand(&runCommandFuture, runCommandInput, experimentsDetails.SubscriptionID, experimentsDetails.ResourceGroup, vmName, experimentsDetails.ScaleSet); err != nil {
					return errors.Errorf("unable to run script on azure instance, err: %v", err)
				}

				// Run probes during chaos
				if len(resultDetails.ProbeDetails) != 0 {
					if err = probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
						return err
					}
				}

				// Wait for Chaos interval
				log.Infof("[Wait]: Waiting for chaos interval of %vs", experimentsDetails.ChaosInterval)
				common.WaitForDuration(experimentsDetails.ChaosInterval)

				log.Infof("[Wait]: Waiting for script execution completion on instance: %v", vmName)
				result, err := runCommand.WaitForRunCommandCompletion(&runCommandFuture, experimentsDetails.SubscriptionID, experimentsDetails.ScaleSet)
				if err != nil {
					return errors.Errorf("%v", err)
				}
				runCommand.GetRunCommandResult(&result)
				if err = azure.CheckRunCommandResultError(&result); err != nil {
					return err
				}
			}
			duration = int(time.Since(ChaosStartTimeStamp).Seconds())
		}
	}
	return nil
}

func abortWatcher(experimentsDetails *experimentTypes.ExperimentDetails, instanceNameList []string, abortRunCommandInput compute.RunCommandInput) {
	<-abort

	log.Info("[Abort]: Chaos Revert Started")
	runCommandFutures := []experimentTypes.RunCommandFuture{}

	for _, vmName := range instanceNameList {
		log.Infof("[Chaos]: Running abort script on the Azure instance: %v", vmName)
		runCommandFuture := experimentTypes.RunCommandFuture{}
		if err := runCommand.PerformRunCommand(&runCommandFuture, abortRunCommandInput, experimentsDetails.SubscriptionID, experimentsDetails.ResourceGroup, vmName, experimentsDetails.ScaleSet); err != nil {
			log.Errorf("unable to run abort script on azure instance, err: %v", err)
		}
		runCommandFutures = append(runCommandFutures, runCommandFuture)
	}

	for i, vmName := range instanceNameList {
		log.Infof("[Wait]: Waiting for abort script execution completion on instance: %v", vmName)
		result, err := runCommand.WaitForRunCommandCompletion(&runCommandFutures[i], experimentsDetails.SubscriptionID, experimentsDetails.ScaleSet)
		if err != nil {
			log.Errorf("%v", err)
		}
		runCommand.GetRunCommandResult(&result)
		if err = azure.CheckRunCommandResultError(&result); err != nil {
			log.Errorf("failed to abort script due to %v", err)
		}
	}
	log.Infof("[Abort]: Chaos Revert Completed")
	os.Exit(1)
}
