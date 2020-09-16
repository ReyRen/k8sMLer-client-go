package k8sMLer_client_go

import (
	"context"
	"flag"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"

	//"k8s.io/client-go/util/retry"
	"log"
	//"os"
	"path/filepath"
	//"text/template/parse"

	//appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	var kubeconfig *string
	var namespaceName, serviceName, labelName string
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional)absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(&namespaceName, "namespace", "", "namespace name")
	if namespaceName == "" {
		log.Fatal("The namespace is required[namespace=]")
	}
	flag.StringVar(&serviceName, "service", "", "service name")
	if serviceName == "" {
		log.Fatal("The service name is required[service=]")
	}
	flag.StringVar(&labelName, "label", "", "label name") // the label name is not required
	flag.Parse()

	// create the client
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	// create svc
	svcClient := clientset.CoreV1().Services(namespaceName)
	service := &apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
			Labels: map[string]string{
				"app": labelName,
			},
			Annotations: nil,
		},
		Spec: apiv1.ServiceSpec{
			Ports: []apiv1.ServicePort{
				{
					Name:        "ssh",
					Protocol:    apiv1.ProtocolTCP,
					AppProtocol: nil,
					Port:        22,
					TargetPort:  intstr.IntOrString{IntVal: 22},
				},
			},
			Selector: map[string]string{
				"app": labelName,
			},
		},
		//Status:     apiv1.ServiceStatus{},
	}

	fmt.Println("creating servide...")
	result, err := svcClient.Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		log.Fatalln("create the service err : ", err)
	}
	fmt.Printf("Created service %q.\n", result.GetObjectMeta().GetName())
}
