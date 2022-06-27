package environment

import (
	"strconv"

	clientTypes "k8s.io/apimachinery/pkg/types"

	experimentTypes "github.com/litmuschaos/litmus-go/pkg/os/network-chaos/types"
	"github.com/litmuschaos/litmus-go/pkg/types"
)

//GetENV fetches all the env variables from the runner pod
func GetENV(experimentDetails *experimentTypes.ExperimentDetails, expName string) {
	experimentDetails.ExperimentName = types.Getenv("EXPERIMENT_NAME", "")
	experimentDetails.ChaosNamespace = types.Getenv("CHAOS_NAMESPACE", "litmus")
	experimentDetails.EngineName = types.Getenv("CHAOSENGINE", "")
	experimentDetails.ChaosDuration, _ = strconv.Atoi(types.Getenv("TOTAL_CHAOS_DURATION", "30"))
	experimentDetails.ChaosInterval, _ = strconv.Atoi(types.Getenv("CHAOS_INTERVAL", "10"))
	experimentDetails.RampTime, _ = strconv.Atoi(types.Getenv("RAMP_TIME", "0"))
	experimentDetails.ChaosLib = types.Getenv("LIB", "litmus")
	experimentDetails.ChaosUID = clientTypes.UID(types.Getenv("CHAOS_UID", ""))
	experimentDetails.InstanceID = types.Getenv("INSTANCE_ID", "")
	experimentDetails.ChaosPodName = types.Getenv("POD_NAME", "")
	experimentDetails.Delay, _ = strconv.Atoi(types.Getenv("STATUS_CHECK_DELAY", "2"))
	experimentDetails.Timeout, _ = strconv.Atoi(types.Getenv("STATUS_CHECK_TIMEOUT", "180"))
	experimentDetails.TargetContainer = types.Getenv("TARGET_CONTAINER", "")
	experimentDetails.NetworkInterfaces = types.Getenv("NETWORK_INTERFACES", "eth0")
	experimentDetails.DestinationHosts = types.Getenv("DESTINATION_HOSTS", "google.com,facebook.com")
	experimentDetails.DestinationIPs = types.Getenv("DESTINATION_IPS", "")
	experimentDetails.AgentEndpoint = types.Getenv("AGENT_ENDPOINT", "")
	experimentDetails.AuthToken = types.Getenv("AUTH_TOKEN", "")

	switch expName {
	case "os-network-loss":
		experimentDetails.NetworkPacketLossPercentage, _ = strconv.Atoi(types.Getenv("NETWORK_PACKET_LOSS_PERCENTAGE", "100"))
	case "os-network-latency":
		experimentDetails.NetworkLatency, _ = strconv.Atoi(types.Getenv("NETWORK_LATENCY", "1000"))
	}

}
