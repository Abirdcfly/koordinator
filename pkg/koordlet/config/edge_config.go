package config

import (
	"os"

	clientcmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

func GenerateKubeClientConfig() *clientcmdv1.Config {
	namedCluster := clientcmdv1.NamedCluster{
		Name: "kubeedge-cluster",
		Cluster: clientcmdv1.Cluster{
			Server:                "https://127.0.0.1:10550",
			InsecureSkipTLSVerify: true,
		},
	}
	namedContext := clientcmdv1.NamedContext{
		Name: "kubeedge-context",
		Context: clientcmdv1.Context{
			Cluster:  "kubeedge-cluster",
			AuthInfo: "edgemesh",
		},
	}
	namedAuthInfo := clientcmdv1.NamedAuthInfo{
		Name: "edgemesh",
		AuthInfo: clientcmdv1.AuthInfo{
			TokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
		},
	}

	return &clientcmdv1.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		Clusters:       []clientcmdv1.NamedCluster{namedCluster},
		Contexts:       []clientcmdv1.NamedContext{namedContext},
		CurrentContext: "kubeedge-context",
		Preferences:    clientcmdv1.Preferences{},
		AuthInfos:      []clientcmdv1.NamedAuthInfo{namedAuthInfo},
	}
}

func SaveKubeConfigFile() error {
	kubeClientConfig := GenerateKubeClientConfig()
	data, err := yaml.Marshal(kubeClientConfig)
	if err != nil {
		return err
	}

	f, err := os.OpenFile("kubeconfig", os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer func() {
		err = f.Close()
		if err != nil {
			klog.ErrorS(err, "close file error")
		}
	}()

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}
