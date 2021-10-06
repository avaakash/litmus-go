package types

import (
	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	clientTypes "k8s.io/apimachinery/pkg/types"
)

// ADD THE ATTRIBUTES OF YOUR CHOICE HERE
// FEW MENDATORY ATTRIBUTES ARE ADDED BY DEFAULT

// ExperimentDetails is for collecting all the experiment-related details
type ExperimentDetails struct {
	ExperimentName     string
	EngineName         string
	ChaosDuration      int
	ChaosInterval      int
	RampTime           int
	ChaosLib           string
	AppNS              string
	AppLabel           string
	AppKind            string
	ChaosUID           clientTypes.UID
	InstanceID         string
	ChaosNamespace     string
	ChaosPodName       string
	Timeout            int
	Delay              int
	TargetContainer    string
	ChaosInjectCmd     string
	ChaosKillCmd       string
	PodsAffectedPerc   int
	TargetPods         string
	LIBImagePullPolicy string
	AzureInstanceNames string
	ResourceGroup      string
	SubscriptionID     string
	ScaleSet           string
	Sequence           string
	HttpChaosType      string
	InstallDependency  string
	OperatingSystem    string
	StreamPort         string
	StreamType         string
	ListenPort         string
	Latency            int
	RateLimit          int
	DataLimit          int
	RequestTimeout     int
	ScriptPath         string
	AbortScriptPath    string
}

type RunCommandFuture struct {
	VmssFuture compute.VirtualMachineScaleSetVMsRunCommandFuture
	VmFuture   compute.VirtualMachinesRunCommandFuture
}
