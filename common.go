package main

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
)

func ParseArg(kubeconfig *string, namespaceName *string, serviceName *string, labelName *string) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		flag.StringVar(kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional)absolute path to the kubeconfig file")
	} else {
		flag.StringVar(kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}

	flag.StringVar(namespaceName, "namespace", "", "namespace name")
	flag.StringVar(serviceName, "service", "", "service name")
	flag.StringVar(labelName, "label", "", "label name") // the label name is not required
	flag.Parse()

	if *namespaceName == "" {
		log.Fatal("The namespace is required[namespace=]")
	}
	if *serviceName == "" {
		log.Fatal("The service name is required[service=]")
	}
}

func CreateClient(clientset **kubernetes.Clientset, kubeconfig *string) error {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		log.Fatal("build config of client err: ", err)
		return err
	}
	*clientset, err = kubernetes.NewForConfig(config)
	return err
}
