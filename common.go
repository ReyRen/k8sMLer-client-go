package main

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"path/filepath"
)

func ParseArg(kubeconfig *string, operator *string, resource *string, namespaceName *string, serviceName *string, labelName *string) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		flag.StringVar(kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional)absolute path to the kubeconfig file")
	} else {
		flag.StringVar(kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(operator, "o", "list", "list,create,delete")
	flag.StringVar(resource, "r", "", "resource type, svc and pod support") // pod, svc
	flag.StringVar(namespaceName, "n", "", "namespace name")
	flag.StringVar(serviceName, "s", "", "service name")
	flag.StringVar(labelName, "l", "", "label name") // the label name is not required
	flag.Parse()

	if *resource == "" {
		log.Fatal("The resource is required[r=], only supports pod and service")
	}
	if *namespaceName == "" {
		log.Fatal("The namespace is required[n=]")
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
