package main

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func CreateClient(clientset **kubernetes.Clientset, kubeconfig *string) error {
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		Error.Printf("build config of client err: %s\n", err)
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

func LogMonitor(c *Client, rd io.Reader, realPvcName *string, nodeNum int, gpuNum int) {
	r := bufio.NewReader(rd)
	flag := 0
	endStr, startStr := PraseTmpString(*realPvcName)

	for {
		line, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			Error.Printf("[%d, %d]:read err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		}
		//fmt.Printf("[%d, %d]:log msgs: %s\n", c.userIds.Uid, c.userIds.Tid, string(line))
		if strings.Contains(string(line), TRAINLOGSTART) || flag != 0 {
			if strings.Contains(string(line), TRAINLOGSTART) {
				exec_init_program(c, startStr+strconv.Itoa(nodeNum-1)+"-pod-"+endStr, nodeNum, gpuNum)
			}
			c.hub.clients[*c.userIds].Head.sm.Type = LOGRESPOND
			c.hub.clients[*c.userIds].Head.sm.Content.Log = string(line)
			c.hub.clients[*c.userIds].Head.logchan <- c.hub.clients[*c.userIds].Head.sm

			if strings.Contains(string(line), TRAINLOGSTART) {
				clientSocket(c, ENDTRAININGSTART)
			} else if strings.Contains(string(line), TRAINLOGERR) {
				clientSocket(c, ENDTRAININGSTOPFAIL)
				resourceOperator(c,
					kubeconfigName,
					"delete",
					"pod",
					nameSpace,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					"10Gi",
					c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
					&c.hub.clients[*c.userIds].Head.rm.realPvcName)
				return
			} else if strings.Contains(string(line), TRAINLOGDONE) {
				clientSocket(c, ENDTRAININGSTOPNORMAL)
				resourceOperator(c,
					kubeconfigName,
					"delete",
					"pod",
					nameSpace,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					"10Gi",
					c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
					&c.hub.clients[*c.userIds].Head.rm.realPvcName)
				return
			}
			/*if strings.Contains(string(line), TRAINLOGSTART) {
				_ = <-c.hub.clients[*c.userIds].Head.signalChan
				// block for tons of msg following Start, then start can't receive
			}*/
			flag = 1
		} else {
			if strings.Contains(string(line), "Connection timed out") {
				c.hub.clients[*c.userIds].Head.sm.Type = RSRESPOND
				//c.hub.clients[*c.userIds].Head.sm.zContent.Log = "FTP: Connection timed out"
				c.hub.clients[*c.userIds].Head.logchan <- c.hub.clients[*c.userIds].Head.sm
			}
			Trace.Printf("[%d, %d]: %s", c.userIds.Uid, c.userIds.Tid, string(line))
			continue
		}
	}
}

func PraseTmpString(tmpString string) (string, string) {
	rm := strings.Split(tmpString, "-")
	return rm[len(rm)-1], rm[0]
}

const (
	// ip and ports with end
	socketServer = "172.18.29.81:8082"
	// ip of mine
	websocketServer = "172.18.29.80:8066"

	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 4096

	// define the default namespace RS located
	nameSpace = "web"
	// define storageclass name
	STORAGECLASS = "web-nfs"

	// statusCode to frontend
	RECVSTART           = 10 // 训练指令已发送
	INSUFFICIENTPENDING = 11 // 没有资源的pending状态
	TRAININGSTART       = 12 // 开始训练
	TRAININGSTOPSUCCESS = 13 // 训练正常结束
	TRAININGSTOPFAILED  = 14 // 训练异常结束
	TRAININGRESET       = 15 // 二度计算失败后，用户重新连入ws，返回给web的标志，否则将一直是INSUFFICIENTPENDING

	TRAINLOGDONE  = "111Done111"
	TRAINLOGSTART = "111Start111"
	TRAINLOGERR   = "111Err111"

	// Type Code
	//STATUSRESPOND = 1
	RSRESPOND  = 2 // ftp超时重连
	LOGRESPOND = 3 // 打印日志

	// Status code for end
	WAITINGRESOURCE       = 4
	RESOURCECOMPLETE      = 5
	ENDTRAININGSTART      = 6
	ENDTRAININGSTOPNORMAL = 7
	ENDTRAININGSTOPFAIL   = 8

	// 10Gi ips substring
	MATCHIPS = "192.168.100."

	// docker image registry server
	REGISTRYSERVER = "172.18.29.81:8080"

	// transfor paramaters (workaround for ssh execute cmd cannot use >)
	PARAMS_TRANS_SCRIPT = "params_trans.sh"
	// init script
	START_SCRIPT = "start.py"

	// POD pvc url
	MOUNTPATH = "/usr/share/horovod/"

	// absoulute of two scripts
	PARAMS_IN_POD = MOUNTPATH + PARAMS_TRANS_SCRIPT
	START_IN_POD  = MOUNTPATH + START_SCRIPT

	// base script URL
	BASE_SCRIPT_URL = "http://192.168.100.1:8008/ftp/script/"

	// two scripts URL
	PARAMS_TRANS_URL      = BASE_SCRIPT_URL + PARAMS_TRANS_SCRIPT
	START_URL             = BASE_SCRIPT_URL + START_SCRIPT
	WGET_PARAMS_TRANS_URL = "wget -c -P " + MOUNTPATH + " " + PARAMS_TRANS_URL + ";"
	WGET_START_URL        = "wget -c -P " + MOUNTPATH + " " + START_URL

	// create pods args
	INIT_TAIL   = "/etc/init.d/ssh start > /dev/null;"
	END_TAIL    = ";tail -f /dev/null"
	MASTER_TAIL = INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + ";python " + START_IN_POD + END_TAIL
	CHILD_TAIL  = INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + END_TAIL

	//images
	IMAGE           = "horovod/horovod:0.19.0-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6-opencv-sk-mplot"
	IMAGE_MMDECTION = "horovod:mmdection"
	// horovod/horovod:0.18.1-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6

	// ftp log
	FTPSERVER = "172.18.29.80:21"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	// lock
	lock sync.Mutex

	newline = []byte{'\n'}
	space   = []byte{' '}

	// log
	Trace   *log.Logger // 记录所有日志
	Info    *log.Logger // 重要的信息
	Warning *log.Logger // 需要注意的信息
	Error   *log.Logger // 非常严重的问题

)

func jsonHandler(data []byte, v interface{}) {
	errJson := json.Unmarshal(data, v)
	if errJson != nil {
		Error.Printf("json err: %s\n", errJson)
	}
}

func getKubeconfigName(kubeconfig *string) {
	if home := homedir.HomeDir(); home != "" { // HomeDir returns the home directory for the current user
		*kubeconfig = filepath.Join(home, ".kube", "config")
	} else {
		*kubeconfig = ""
		Warning.Println("The kubeconfig is null")
	}
}

func get_node_info(c *Client) {
	/*var capacity []byte
	var used []byte

	capacityResult := exec.Command("/bin/bash", "-c", `kubectl describe nodes  |  tr -d '\000' | sed -n -e '/^Name/,/Roles/p' -e '/^Capacity/,/Allocatable/p' -e '/^Allocated resources/,/Events/p' | grep -e Name  -e  nvidia.com  | perl -pe 's/\n//'  |  perl -pe 's/Name:/\n/g' | sed 's/nvidia.com\/gpu:\?//g'  | sed '1s/^/Node Available(GPUs)  Used(GPUs)/' | sed 's/$/0 0 0/'  | awk '{print $1, $2, $3}'  | column -t | awk '{sum += $2};END {print sum}'`)
	usedResult := exec.Command("/bin/bash", "-c", `kubectl describe nodes  |  tr -d '\000' | sed -n -e '/^Name/,/Roles/p' -e '/^Capacity/,/Allocatable/p' -e '/^Allocated resources/,/Events/p' | grep -e Name  -e  nvidia.com  | perl -pe 's/\n//'  |  perl -pe 's/Name:/\n/g' | sed 's/nvidia.com\/gpu:\?//g'  | sed '1s/^/Node Available(GPUs)  Used(GPUs)/' | sed 's/$/0 0 0/'  | awk '{print $1, $2, $3}'  | column -t | awk '{sum += $3};END {print sum}'`)
	capacity, _ = capacityResult.Output()
	used, _ = usedResult.Output()*/

	// assemble send data
	// dont set the type

	getKubeconfigName(&kubeconfigName) // fill up into the kubeconfig

	// create k8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfigName)
	if err != nil {
		Error.Printf("[%d, %d]: CreateClient err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}

	//var list *apiv1.NodeList
	list, _ := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		TypeMeta:      metav1.TypeMeta{},
		LabelSelector: "accelerator",
	})
	nodeList := list.Items

	var nodeLabels string
	var nodeNames string
	var nodeStatus string

	for _, node := range nodeList {
		nodeStatusTmp := node.Status.Conditions
		for _, readyType := range nodeStatusTmp {
			if readyType.Type == apiv1.NodeReady {
				if readyType.Status == apiv1.ConditionTrue {
					nodeStatus += "online"
					nodeStatus += ","
				} else if readyType.Status == apiv1.ConditionFalse {
					nodeStatus += "offline"
					nodeStatus += ","
				} else {
					nodeStatus += "offline"
					nodeStatus += ","
				}
			}
		}

		tmpLabel := node.GetLabels()
		nodeLabels += tmpLabel["accelerator"]
		nodeLabels += ","
		nodeNames += node.GetName()
		nodeNames += ","
	}
	c.hub.clients[*c.userIds].Head.sm.Content.ResourceInfo.NodesListerName = nodeNames
	c.hub.clients[*c.userIds].Head.sm.Content.ResourceInfo.NodesListerLabel = nodeLabels
	c.hub.clients[*c.userIds].Head.sm.Content.ResourceInfo.NodesListerStatus = nodeStatus
}

func exec_init_program(c *Client, exec_pod_name string, nodeNum int, gpuNum int) {
	var base_cmd_string string
	if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 7 {
		base_cmd_string = "kubectl exec " +
			exec_pod_name +
			" -n " +
			nameSpace +
			" -it -- " +
			"/bin/bash " + PARAMS_IN_POD + " \"" +
			"--ip=" +
			c.hub.clients[*c.userIds].Head.ips +
			" --nodes=" +
			strconv.Itoa(nodeNum) +
			" --model_parameters=" +
			c.hub.clients[*c.userIds].Head.rm.Content.Params +
			" --mp_size=" +
			strconv.Itoa(gpuNum) +
			" --user_id=" +
			strconv.Itoa(c.userIds.Uid) +
			" --task_id=" +
			strconv.Itoa(c.userIds.Tid) +
			" --model_type=" +
			strconv.Itoa(c.hub.clients[*c.userIds].Head.rm.Content.ModelType) +
			" --cmd=" +
			"'" +
			c.hub.clients[*c.userIds].Head.rm.Content.CommandBox +
			"'" +
			"\""
	} else {
		base_cmd_string = "kubectl exec " +
			exec_pod_name +
			" -n " +
			nameSpace +
			" -it -- " +
			"/bin/bash " + PARAMS_IN_POD + " \"" +
			"--ip=" +
			c.hub.clients[*c.userIds].Head.ips +
			" --nodes=" +
			strconv.Itoa(nodeNum) +
			" --model_parameters=" +
			c.hub.clients[*c.userIds].Head.rm.Content.Params +
			" --mp_size=" +
			strconv.Itoa(gpuNum) +
			" --user_id=" +
			strconv.Itoa(c.userIds.Uid) +
			" --task_id=" +
			strconv.Itoa(c.userIds.Tid) +
			" --model_type=" +
			strconv.Itoa(c.hub.clients[*c.userIds].Head.rm.Content.ModelType) +
			" --model_url=" +
			c.hub.clients[*c.userIds].Head.rm.Content.OriginalModelUrl +
			" --model_url_con=" +
			c.hub.clients[*c.userIds].Head.rm.Content.ContinuousModelUrl +
			" --framework=" +
			strconv.Itoa(c.hub.clients[*c.userIds].Head.rm.Content.FrameworkType) +
			" --selected_dataset=" +
			c.hub.clients[*c.userIds].Head.rm.Content.SelectedDataset +
			"\""
	}

	cmd := exec.Command("/bin/bash", "-c", base_cmd_string)
	Trace.Println(cmd)
	if err := cmd.Run(); err != nil {
		Error.Printf("[%d, %d]: command run err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
}

func log_back_to_frontend(c *Client,
	kubeconfig string,
	namespaceName string,
	realPvcName *string,
	nodeNum int,
	gpuNum int) {

	getKubeconfigName(&kubeconfig) // fill up into the kubeconfig

	// createk8s-client
	var clientset *kubernetes.Clientset
	err := CreateClient(&clientset, &kubeconfig)
	if err != nil {
		Error.Printf("[%d, %d]: CreateClient err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
	// define resource
	podClient := clientset.CoreV1().Pods(namespaceName)

	endStr, startStr := PraseTmpString(*realPvcName)
	//fmt.Println("get pods log...")
	result := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+endStr, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: false, // timestamps
	})
	// used for ftp log upload
	result2 := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+endStr, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: false, // timestamps
	})
	podLogs, err := result.Stream(context.TODO())
	if err != nil {
		Error.Println("[%d, %d]: podLogs err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	podLogs2, err := result2.Stream(context.TODO())
	if err != nil {
		Error.Println("[%d, %d]: podLogs2 err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	//return podLogs
	defer podLogs.Close()
	go ftpUploader(c, podLogs2)
	LogMonitor(c, podLogs, realPvcName, nodeNum, gpuNum)
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func clientSocket(c *Client, statusCode int) {
	// create socket with end
	conn, err := net.Dial("tcp", socketServer)
	if err != nil {
		Error.Printf("[%d, %d]: clientSocket err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	defer conn.Close()
	var clientmsg clientsocketmsg
	clientmsg.Uid = c.userIds.Uid
	clientmsg.Tid = c.userIds.Tid
	clientmsg.StatusId = statusCode
	socketmsg, _ := json.Marshal(clientmsg)
	_, err = conn.Write(socketmsg)
	if err != nil {
		Error.Printf("[%d, %d]: clientSocket send err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
}

func trimRsNum(selectNodes *[]selectNodes, nodeNum *int, gpuNum *int) {
	*nodeNum = len(*selectNodes)
	for _, v := range *selectNodes {
		*gpuNum = v.GPUNum
		break
	}
}
