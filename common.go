package main

import (
	"flag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"path/filepath"
	"time"
)

func ParseArg(kubeconfig *string, operator *string, resource *string, namespaceName *string, kindName *string, labelName *string, gpuQuantity *int64) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		flag.StringVar(kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional)absolute path to the kubeconfig file")
	} else {
		flag.StringVar(kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(operator, "o", "list", "list,create,delete")
	flag.StringVar(resource, "r", "", "service,pod") // pod, svc
	flag.StringVar(namespaceName, "n", "", "namespaceName")
	flag.StringVar(kindName, "k", "", "kindName[svcName, podName]")
	flag.StringVar(labelName, "l", "", "labelName") // the label name is not required
	flag.Int64Var(gpuQuantity, "g", int64(0), "gpu quantities[0,1,2..]")
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

func GetRandomString(l int) string {
	str := "0123456789abcefghijklmnopqrstuvwxyz"
	bytes := []byte(str)
	var result []byte
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < l; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}
