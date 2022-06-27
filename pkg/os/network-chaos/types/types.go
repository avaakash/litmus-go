package types

import (
	clientTypes "k8s.io/apimachinery/pkg/types"
)

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	ExperimentName              string
	EngineName                  string
	ChaosDuration               int
	ChaosInterval               int
	RampTime                    int
	ChaosLib                    string
	ChaosUID                    clientTypes.UID
	InstanceID                  string
	ChaosNamespace              string
	ChaosPodName                string
	Timeout                     int
	Delay                       int
	TargetContainer             string
	LIBImagePullPolicy          string
	NetworkInterfaces           string
	DestinationHosts            string
	DestinationIPs              string
	NetworkLatency              int
	NetworkPacketLossPercentage int
	AgentEndpoint               string
	AuthToken                   string
}

type NetworkChaosParams struct {
	ExperimentName              string
	NetworkLatency              int
	NetworkPacketLossPercentage int
	DestinationHosts            string
	DestinationIPs              string
	NetworkInterface            string
}
