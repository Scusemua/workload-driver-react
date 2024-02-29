package config

import (
	"encoding/json"
	"flag"
	"path/filepath"

	"k8s.io/client-go/util/homedir"
)

const (
	OptionName    = "name"
	OptionDefault = "default"
	OptionDesc    = "description"
)

type Configuration struct {
	SpoofCluster            bool   `yaml:"spoof-gateway" json:"spoof-gateway" description:"If true, use the fake cluster."`
	InCluster               bool   `yaml:"in-cluster" json:"in-cluster" description:"Should be true if running from within the kubernetes cluster."`
	KernelQueryInterval     string `yaml:"kernel-query-interval" json:"kernel-query-interval" default:"5s" description:"How frequently to query the Cluster for updated kernel information."`
	NodeQueryInterval       string `yaml:"node-query-interval" json:"node-query-interval" default:"10s" description:"How frequently to query the Cluster for updated Kubernetes node information."`
	KernelSpecQueryInterval string `yaml:"kernel-spec-query-interval" json:"kernel-spec-query-interval" default:"600s" description:"How frequently to query the Cluster for updated Jupyter kernel spec information."`
	KubeConfig              string `yaml:"kubeconfig" json:"kubeconfig" description:"Absolute path to the kubeconfig file."`
	GatewayAddress          string `yaml:"gateway-address" json:"gateway-address" description:"The IP address that the front-end should use to connect to the Gateway."`
	JupyterServerAddress    string `yaml:"jupyter-server-address" json:"jupyter-server-address" description:"The IP address of the Jupyter Server."`
	ServerPort int `yaml:"server-port" json:"server-port" description:"Port of the backend server."`

	Valid bool `json:"Valid"` // Used to determine if the struct was sent/received correctly over the network.
}

func (c *Configuration) String() string {
	out, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

func GetConfiguration() *Configuration {
	var spoofFlag = flag.Bool("spoof-cluster", true, "Spoof the connection to the Cluster Gateway.")
	var inClusterFlag = flag.Bool("in-cluster", false, "Should be true if running from within the kubernetes cluster.")
	var kernelQueryIntervalFlag = flag.String("kernel-query-interval", "60s", "How often to refresh kernels from Cluster Gateway.")
	var nodeQueryIntervalFlag = flag.String("node-query-interval", "120s", "How often to refresh nodes from Cluster Gateway.")
	var gatewayAddressFlag = flag.String("gateway-address", "localhost:9990", "The IP address that the front-end should use to connect to the Gateway.")
	var kernelSpecQueryIntervalFlag = flag.String("kernel-spec-query-interval", "600s", "How frequently to query the Cluster for updated Jupyter kernel spec information.")
	var jupyterServerAddressFlag = flag.String("jupyter-server-address", "http://localhost:8888", "The IP address of the Jupyter Server.")
	var serverPortFlag = flag.Int("server-port", 8000, "Port of the backend server.")

	var kubeconfigFlag *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfigFlag = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfigFlag = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.Parse()

	return &Configuration{
		SpoofCluster:            *spoofFlag,
		InCluster:               *inClusterFlag,
		KernelQueryInterval:     *kernelQueryIntervalFlag,
		NodeQueryInterval:       *nodeQueryIntervalFlag,
		KubeConfig:              *kubeconfigFlag,
		GatewayAddress:          *gatewayAddressFlag,
		KernelSpecQueryInterval: *kernelSpecQueryIntervalFlag,
		JupyterServerAddress:    *jupyterServerAddressFlag,
		ServerPort: *serverPortFlag,
		Valid:                   true,
	}
}

// func GetOptions() *Configuration {
// 	var yamlPath string
// 	flag.StringVar(&yamlPath, "config", "config.yaml", "Path to the YAML configuration file.")
// 	flag.Parse()

// 	yamlFile, err := os.ReadFile(yamlPath)
// 	if err != nil {
// 		log.Printf("[ERROR] Failed to read YAML config file \"%s\": %v\n.", yamlPath, err)
// 		log.Printf("Using default configuration file instead.")
// 		return GetDefaultConfiguration()
// 	}

// 	var conf Configuration
// 	err = yaml.Unmarshal(yamlFile, &conf)
// 	if err != nil {
// 		log.Fatalf("Unmarshal: %v", err)
// 	}

// 	return &conf
// }
