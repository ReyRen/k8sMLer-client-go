package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func ParseArg(kubeconfig *string, operator *string, resource *string, namespaceName *string, kindName *string, labelName *string, gpuQuantity *int64, cap *string) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		flag.StringVar(kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional)absolute path to the kubeconfig file")
	} else {
		flag.StringVar(kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.StringVar(operator, "o", "list", "list,create,delete,log")
	flag.StringVar(resource, "r", "", "service,pod,pvc") // pod, svc
	flag.StringVar(namespaceName, "n", "", "namespaceName")
	flag.StringVar(kindName, "k", "", "kindName[svcName, podName]")
	flag.StringVar(labelName, "l", "", "labelName") // the label name is not required
	flag.Int64Var(gpuQuantity, "g", int64(0), "gpu quantities[0,1,2..]")
	flag.StringVar(cap, "c", "10Gi", "pvc volume capacity[10Gi,20Gi,30Gi,40Gi,50Gi]")
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

func LogMonitor(c *Client, rd io.Reader) {
	r := bufio.NewReader(rd)

	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			//time.Sleep(500 * time.Millisecond)
			break
		} else if err != nil {
			log.Fatalln("read err: ", err)
		}
		//func() {
		//os.Stdout.Write(line)
		if c.addr != "" {
			c.hub.clients[*c.userIds].Head.sm.Type = LOGRESPOND
			c.hub.clients[*c.userIds].Head.sm.Content.Log = string(line)
			c.hub.broadcast <- c
		}
		//}()
	}
}

func PraseTmpString(tmpString string) (string, string) {
	rm := strings.Split(tmpString, "-")
	return rm[len(rm)-1], rm[0]
}

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 512

	// define the default namespace RS located
	nameSpace = "web"

	// log for flushed client
	NOTLOGGED = 10
	LOGSTART  = 11

	// statusCode to frontend
	RECVSTART           = 10 // click start
	RECVSTOP            = 11 // click stop
	TRAININGSTART       = 12 // after rs ready and code running
	TRAININGSTOPSUCCESS = 13 // success finished
	TRAININGSTOPFAILED  = 14 // error finished

	TRAININGLOGDONE  = "Done\n"
	TRAININGLOGSTART = "Start\n"
	TRAININGLOGERR   = "Err\n"

	// Type Code
	STATUSRESPOND   = 1
	RESOURCERESPOND = 2
	LOGRESPOND      = 3
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

func jsonHandler(data []byte, v interface{}) {
	errJson := json.Unmarshal(data, v)
	if errJson != nil {
		log.Fatalln("json err: ", errJson)
	}
}

func getKubeconfigName(kubeconfig *string) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		*kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		*kubeconfig = ""
		fmt.Println("The kubeconfig is null")
	}
}

func set_gpu_rest(c *Client) {
	var capacity []byte
	var used []byte

	capacityResult := exec.Command("/bin/bash", "-c", `kubectl describe nodes  |  tr -d '\000' | sed -n -e '/^Name/,/Roles/p' -e '/^Capacity/,/Allocatable/p' -e '/^Allocated resources/,/Events/p' | grep -e Name  -e  nvidia.com  | perl -pe 's/\n//'  |  perl -pe 's/Name:/\n/g' | sed 's/nvidia.com\/gpu:\?//g'  | sed '1s/^/Node Available(GPUs)  Used(GPUs)/' | sed 's/$/0 0 0/'  | awk '{print $1, $2, $3}'  | column -t | awk '{sum += $2};END {print sum}'`)
	usedResult := exec.Command("/bin/bash", "-c", `kubectl describe nodes  |  tr -d '\000' | sed -n -e '/^Name/,/Roles/p' -e '/^Capacity/,/Allocatable/p' -e '/^Allocated resources/,/Events/p' | grep -e Name  -e  nvidia.com  | perl -pe 's/\n//'  |  perl -pe 's/Name:/\n/g' | sed 's/nvidia.com\/gpu:\?//g'  | sed '1s/^/Node Available(GPUs)  Used(GPUs)/' | sed 's/$/0 0 0/'  | awk '{print $1, $2, $3}'  | column -t | awk '{sum += $3};END {print sum}'`)
	capacity, _ = capacityResult.Output()
	used, _ = usedResult.Output()

	// assemble send data
	// dont set the type
	c.hub.clients[*c.userIds].Head.sm.Content.GpuInfo.GpuCapacity = string(capacity)
	c.hub.clients[*c.userIds].Head.sm.Content.GpuInfo.GpuUsed = string(used)
}

func log_back_to_frontend(c *Client, kubeconfig string, namespaceName string, nodeQuantity int, realPvcName *string) {

	getKubeconfigName(&kubeconfig) // fill up into the kubeconfig

	// createk8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		panic(err)
	}
	// define resource
	podClient := clientset.CoreV1().Pods(namespaceName)

	endStr, startStr := PraseTmpString(*realPvcName)
	//fmt.Println("get pods log...")
	result := podClient.GetLogs(startStr+strconv.Itoa(nodeQuantity-1)+"-pod-"+endStr, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: true, // timestamps
	})
	podLogs, err := result.Stream(context.TODO())
	if err != nil {
		log.Fatalln("podLogs stream err : ", err)
	}
	//return podLogs
	defer podLogs.Close()
	if c.addr != "" {
		LogMonitor(c, podLogs)
	}
}

func (c *Client) logDisplay() {
	for true {
		select {
		case logFlag := <-c.hub.clients[*c.userIds].Head.logChan:
			if logFlag == LOGSTART {
				//c.hub.clients[*c.userIds].Head.sm.Type = 3
				if c.addr != "" {
					log_back_to_frontend(c, kubeconfigName, nameSpace, c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes, &c.hub.clients[*c.userIds].Head.rm.realPvcName)
				}
			}
		}
	}
}
