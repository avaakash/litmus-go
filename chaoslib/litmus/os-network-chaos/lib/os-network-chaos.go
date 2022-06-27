package lib

import (
	"time"

	clients "github.com/litmuschaos/litmus-go/pkg/clients"
	"github.com/litmuschaos/litmus-go/pkg/events"
	"github.com/litmuschaos/litmus-go/pkg/log"
	"github.com/litmuschaos/litmus-go/pkg/machine/common/messages"
	experimentTypes "github.com/litmuschaos/litmus-go/pkg/os/network-chaos/types"
	"github.com/litmuschaos/litmus-go/pkg/probe"
	"github.com/litmuschaos/litmus-go/pkg/types"
	"github.com/litmuschaos/litmus-go/pkg/utils/common"
	"github.com/pkg/errors"
)

// PrepareOSNetworkChaos contains the preparation and injection steps for the experiment
func PrepareOSNetworkChaos(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	// waiting for the ramp time before chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time before injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	if err := injectChaos(experimentsDetails, clients, resultDetails, eventsDetails, chaosDetails); err != nil {
		return err
	}
	// wait for the ramp time after chaos injection
	if experimentsDetails.RampTime != 0 {
		log.Infof("[Ramp]: Waiting for the %vs ramp time after injecting chaos", experimentsDetails.RampTime)
		common.WaitForDuration(experimentsDetails.RampTime)
	}

	return nil
}

// injectChaos injects network chaos
func injectChaos(experimentsDetails *experimentTypes.ExperimentDetails, clients clients.ClientSets, resultDetails *types.ResultDetails, eventsDetails *types.EventDetails, chaosDetails *types.ChaosDetails) error {

	//ChaosStartTimeStamp contains the start timestamp, when the chaos injection begin
	ChaosStartTimeStamp := time.Now()
	duration := int(time.Since(ChaosStartTimeStamp).Seconds())

	for duration < experimentsDetails.ChaosDuration {

		if experimentsDetails.EngineName != "" {
			msg := "Injecting " + experimentsDetails.ExperimentName + " chaos in VM instance"
			types.SetEngineEventAttributes(eventsDetails, types.ChaosInject, msg, "Normal", chaosDetails)
			events.GenerateEvents(eventsDetails, clients, chaosDetails, "ChaosEngine")
		}

		timeDuration := 60 * time.Second

		log.Infof("[Chaos]: Injecting network chaos on interface %v ", experimentsDetails.NetworkInterfaces)

		// prepare payload to send
		networkChaosPayload := experimentTypes.NetworkChaosParams{
			ExperimentName:              "os-network-latency",
			NetworkLatency:              experimentsDetails.NetworkLatency,
			NetworkPacketLossPercentage: experimentsDetails.NetworkPacketLossPercentage,
			DestinationHosts:            experimentsDetails.DestinationHosts,
			DestinationIPs:              "experimentsDetails.DestinationIPs",
			NetworkInterface:            experimentsDetails.NetworkInterfaces,
		}

		feedback, payload, err := messages.SendMessageToAgent(chaosDetails.WebsocketConnections[0], "EXECUTE_EXPERIMENT", networkChaosPayload, &timeDuration)
		if err != nil {
			return errors.Errorf("failed to send message to agent, err: %v", err)
		}
		log.Infof("feedback: %v", feedback)
		// ACTION_SUCCESSFUL feedback is received only if the process is killed successfully
		if feedback != "ACTION_SUCCESSFUL" {
			if feedback == "ERROR" {

				agentError, err := messages.GetErrorMessage(payload)
				if err != nil {
					return errors.Errorf("failed to interpret error message from agent, err: %v", err)
				}

				return errors.Errorf("error occured while injecting network chaos on interface %v, err: %s", experimentsDetails.NetworkInterfaces, agentError)
			}

			return errors.Errorf("unintelligible feedback received from agent: %s", feedback)
		}

		log.Info("[Chaos]: network chaos injected successfully")

		// run the probes during chaos
		// the OnChaos probes execution will start in the first iteration and keep running for the entire chaos duration
		if len(resultDetails.ProbeDetails) != 0 {
			if err = probe.RunProbes(chaosDetails, clients, resultDetails, "DuringChaos", eventsDetails); err != nil {
				return err
			}
		}

		// wait for the chaos interval
		// log.Infof("[Wait]: Waiting for chaos interval of %vs", experimentsDetails.ChaosInterval)
		// if err := common.WaitForDurationAndCheckLiveness(chaosDetails.WebsocketConnections, []string{experimentsDetails.AgentEndpoint}, experimentsDetails.ChaosInterval, nil, nil); err != nil {
		// 	return errors.Errorf("error occurred during liveness check, err: %v", err)
		// }

		// wait for the chaos interval
		log.Infof("[Wait]: Waiting for chaos interval of %vs", experimentsDetails.ChaosInterval)
		common.WaitForDuration(experimentsDetails.ChaosInterval)

		feedback, payload, err = messages.SendMessageToAgent(chaosDetails.WebsocketConnections[0], "REVERT_EXPERIMENT", networkChaosPayload, &timeDuration)
		if err != nil {
			return errors.Errorf("failed to send message to agent, err: %v", err)
		}

		// ACTION_SUCCESSFUL feedback is received only if the process is killed successfully
		if feedback != "ACTION_SUCCESSFUL" {
			if feedback == "ERROR" {

				agentError, err := messages.GetErrorMessage(payload)
				if err != nil {
					return errors.Errorf("failed to interpret error message from agent, err: %v", err)
				}
				return errors.Errorf("error occured while injecting network chaos on interface %v, err: %s", experimentsDetails.NetworkInterfaces, agentError)
			}

			return errors.Errorf("unintelligible feedback received from agent: %s", feedback)
		}

		duration = int(time.Since(ChaosStartTimeStamp).Seconds())
	}

	return nil
}
