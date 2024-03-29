package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"math/rand"
	"net"
	"os"
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

//func LogMonitor(c *Client, rd io.Reader, startStr string, RandomName string, nodeNum int, gpuNum int) {
//func LogMonitor(c *Client, rr *rest.Request, startStr string, RandomName string, nodeNum int, gpuNum int) {
func LogMonitor(c *Client, podClient v1.PodInterface, startStr string, RandomName string, nodeNum int, gpuNum int) {
	flag := 0
	/*
		TODO: workaround for stream send EOF after 4 hours
	*/
AGAIN:
	result := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+RandomName, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: false, // timestamps
	})
	podLogs, err := result.Stream(context.TODO())
	if err != nil {
		Error.Printf("[%d, %d]: podLogs err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}
	defer podLogs.Close()
	r := bufio.NewReader(podLogs)
	//endStr, startStr := PraseTmpString(*realPvcName)

	for {
		line, err := r.ReadBytes('\n')
		if err == io.ErrClosedPipe {
			Trace.Println("io.ErrClosedPipe:", err)
			goto AGAIN
		} else if err == io.ErrNoProgress {
			Trace.Println("io.ErrNoProgress:", err)
			goto AGAIN
		} else if err == io.ErrShortBuffer {
			Trace.Println("io.ErrShortBuffer:", err)
			goto AGAIN
		} else if err == io.ErrShortWrite {
			Trace.Println("io.ErrShortWrite:", err)
			goto AGAIN
		} else if err == io.ErrUnexpectedEOF {
			Trace.Println("io.ErrUnexpectedEOF:", err)
			goto AGAIN
		} else if err == io.EOF {
			//break
			Trace.Printf("[%d, %d]: LogMonitor stream get io.EOF: %s\n", c.userIds.Uid, c.userIds.Tid, err)
			podLogs.Close()
			goto AGAIN
		} else if err != nil {
			Error.Printf("[%d, %d]:read err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
			break
		}
		//	Trace.Printf("[%d, %d]:log msgs: %s\n", c.userIds.Uid, c.userIds.Tid, string(line))
		if strings.Contains(string(line), TRAINLOGSTART) || flag != 0 {
			if strings.Contains(string(line), TRAINLOGSTART) {
				exec_init_program(c, startStr+strconv.Itoa(nodeNum-1)+"-pod-"+RandomName, nodeNum, gpuNum)
			}
			c.hub.clients[*c.userIds].Head.sm.Type = LOGRESPOND
			c.hub.clients[*c.userIds].Head.sm.Content.Log = string(line)
			c.hub.clients[*c.userIds].Head.logchan <- c.hub.clients[*c.userIds].Head.sm

			if strings.Contains(string(line), TRAINLOGSTART) {
				clientSocket(c, ENDTRAININGSTART)
			} else if strings.Contains(string(line), TRAINLOGERR) {
				clientSocket(c, ENDTRAININGSTOPFAIL)
				c.hub.clients[*c.userIds].Head.sm.Type = LOGRESPOND
				c.hub.clients[*c.userIds].Head.sm.Content.Log = string(line)
				c.hub.clients[*c.userIds].Head.logchan <- c.hub.clients[*c.userIds].Head.sm
				resourceOperator(c,
					kubeconfigName,
					"delete",
					"pod",
					nameSpace,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					"10Gi",
					c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
					c.hub.clients[*c.userIds].Head.rm.RandomName)
				return
			} else if strings.Contains(string(line), TRAINLOGDONE) {
				clientSocket(c, ENDTRAININGSTOPNORMAL)
				c.hub.clients[*c.userIds].Head.sm.Type = LOGRESPOND
				c.hub.clients[*c.userIds].Head.sm.Content.Log = string(line)
				c.hub.clients[*c.userIds].Head.logchan <- c.hub.clients[*c.userIds].Head.sm
				resourceOperator(c,
					kubeconfigName,
					"delete",
					"pod",
					nameSpace,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
					"10Gi",
					c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes,
					c.hub.clients[*c.userIds].Head.rm.RandomName)
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
	DATA_WEB_SERVER = "172.18.29.81"

	// ip and ports with end
	socketServer = DATA_WEB_SERVER + ":8020"
	// ip of mine
	websocketServer = "172.18.29.80:8066"

	// Time allowed to write a message to the peer.
	writeWait = 60000000 * time.Second
	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second
	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
	// Maximum message size allowed from peer.
	maxMessageSize = 4096

	// define the default namespace RS located
	nameSpace = "generalai"
	// define storageclass name
	STORAGECLASS = "generalai-nfs-storageclass"

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

	MATCHIPS = "192.168.100."

	// Status code for end
	WAITINGRESOURCE       = 4
	RESOURCECOMPLETE      = 5
	ENDTRAININGSTART      = 6
	ENDTRAININGSTOPNORMAL = 7
	ENDTRAININGSTOPFAIL   = 8

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
	BASE_SCRIPT_URL = "http://172.18.29.81/script/"

	// two scripts URL
	PARAMS_TRANS_URL      = BASE_SCRIPT_URL + PARAMS_TRANS_SCRIPT
	START_URL             = BASE_SCRIPT_URL + START_SCRIPT
	WGET_PARAMS_TRANS_URL = ";wget -c -P " + MOUNTPATH + " " + PARAMS_TRANS_URL + ";"
	WGET_START_URL        = "wget -c -P " + MOUNTPATH + " " + START_URL

	// create pods args
	INIT_TAIL = "/etc/init.d/ssh start > /dev/null"
	END_TAIL  = ";tail -f /dev/null"
	//MASTER_TAIL = INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + ";python " + START_IN_POD + END_TAIL
	//CHILD_TAIL  = INIT_TAIL + WGET_PARAMS_TRANS_URL + WGET_START_URL + END_TAIL

	//images
	//IMAGE           = "horovod/horovod:0.19.0-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6-opencv-sk-mplot"
	IMAGE           = "172.18.29.81:8080/test-images/tf1.14.0_torch1.2.0_py3.6_horovod0.19_opencv_sk_mplot_gjx:02-07"
	IMAGE_MMDECTION = "horovod:mmdection"
	// horovod/horovod:0.18.1-tf1.14.0-torch1.2.0-mxnet1.5.0-py3.6

	// ftp log
	FTPSERVER = DATA_WEB_SERVER + ":21"

	MOD_UPDATE = "update"
)

const (
	BEFORECREATE = 0
	CREATING     = 1
	POSTCREATE   = 2
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	// 切片队列
	QUEUELIST []*headNode

	UPDATEMAP map[string][]string

	IP_POOL map[string]bool
)

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
	//if c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 7 || c.hub.clients[*c.userIds].Head.rm.Content.ModelType == 6 {
	base_cmd_string = "kubectl exec " +
		exec_pod_name +
		" -n " +
		nameSpace +
		" -it -- " +
		"/bin/bash " + "/storage-root/scripts/params_trans.sh" + " \"" +
		" --ip=" +
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
		" --distributingMethod=" +
		strconv.Itoa(c.hub.clients[*c.userIds].Head.rm.Content.DistributingMethod) +
		" --modelName=" +
		c.hub.clients[*c.userIds].Head.rm.Content.ModelName +
		" --model_url=" +
		c.hub.clients[*c.userIds].Head.rm.Content.OriginalModelUrl +
		"\""
	/*} else {
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
	}*/

	cmd := exec.Command("/bin/bash", "-c", base_cmd_string)
	//Trace.Println(cmd)
	if err := cmd.Run(); err != nil {
		Error.Printf("[%d, %d]: command run err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
}

func log_back_to_frontend(c *Client,
	kubeconfig string,
	namespaceName string,
	startStr string,
	RandomName string,
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

	//endStr, startStr := PraseTmpString(*realPvcName)
	//fmt.Println("get pods log...")
	/*result := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+RandomName, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: false, // timestamps
	})*/
	// used for ftp log upload
	result2 := podClient.GetLogs(startStr+strconv.Itoa(nodeNum-1)+"-pod-"+RandomName, &apiv1.PodLogOptions{
		Container:  "",
		Follow:     true,
		Previous:   false,
		Timestamps: false, // timestamps
	})
	/*podLogs, err := result.Stream(context.TODO())
	if err != nil {
		Error.Println("[%d, %d]: podLogs err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}*/
	/*podLogs2, err := result2.Stream(context.TODO())
	if err != nil {
		Error.Println("[%d, %d]: podLogs2 err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
		return
	}*/
	//return podLogs
	//defer podLogs.Close()
	//go ftpUploader(c, podLogs2)
	go ftpUploader(c, result2)
	//LogMonitor(c, podLogs, startStr, RandomName, nodeNum, gpuNum)
	//LogMonitor(c, result, startStr, RandomName, nodeNum, gpuNum)
	LogMonitor(c, podClient, startStr, RandomName, nodeNum, gpuNum)
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
	if statusCode == RESOURCECOMPLETE {
		clientmsg.PodName = c.hub.clients[*c.userIds].Head.rm.RandomName
	} else {
		clientmsg.PodName = ""
	}
	socketmsg, _ := json.Marshal(clientmsg)
	_, err = conn.Write(socketmsg)
	if err != nil {
		Error.Printf("[%d, %d]: clientSocket send err: %s\n", c.userIds.Uid, c.userIds.Tid, err)
	}
	c.recordToUpdate(statusCode)
}

func trimRsNum(selectNodes *[]selectNodes, nodeNum *int, gpuNum *int) {
	*nodeNum = len(*selectNodes)
	for _, v := range *selectNodes {
		*gpuNum = v.GPUNum
		break
	}
}

/* used to update */
func (c *Client) recordToUpdate(statusCode int) {

	mapKey := strconv.Itoa(c.userIds.Uid) + "-" + strconv.Itoa(c.userIds.Tid)

	file, error := os.OpenFile(".update", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0766)
	if error != nil {
		fmt.Println(error)
	}

	defer file.Close()
	if _, ok := UPDATEMAP[mapKey]; ok {
		delete(UPDATEMAP, mapKey)
	}

	//handle map {"4-129":["zz8k5nsfzv0jljl","2","node3-1,node2-1,node2-1","gpu","8","0"]}
	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], c.hub.clients[*c.userIds].Head.rm.RandomName)           //0
	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], c.hub.clients[*c.userIds].Head.rm.FtpFileName)          //1
	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], strconv.Itoa(c.hub.clients[*c.userIds].Head.sm.Type))   //2
	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], c.hub.clients[*c.userIds].Head.rm.Content.ResourceType) //3

	var selectedNodes []string
	for _, v := range *(c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes) {
		selectedNodes = append(selectedNodes, v.NodeNames+"|"+strconv.Itoa(v.GPUNum))
	}
	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], strings.Join(selectedNodes, ",")) //4

	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], strconv.Itoa(statusCode)) // socket statusId //5

	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], "") // updated:"", not updated:"1" //6

	UPDATEMAP[mapKey] = append(UPDATEMAP[mapKey], c.hub.clients[*c.userIds].Head.ips) // 7

	//dataReady, err := json.MarshalIndent(UPDATEMAP, "", " ")
	dataReady, err := json.Marshal(UPDATEMAP)
	if err != nil {
		Trace.Printf("recordToUpdate MarshalIndent err: %s\n", err)
	}
	file.Write(dataReady)
}
func (c *Client) removeToUpdate() {
	mapKey := strconv.Itoa(c.userIds.Uid) + "-" + strconv.Itoa(c.userIds.Tid)
	delete(UPDATEMAP, mapKey)

	file, error := os.OpenFile(".update", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0766)
	defer file.Close()
	if error != nil {
		fmt.Println(error)
	}
	//dataReady, err := json.MarshalIndent(UPDATEMAP, "", " ")
	dataReady, err := json.Marshal(UPDATEMAP)
	if err != nil {
		Trace.Printf("recordToUpdate MarshalIndent err: %s\n", err)
	}
	file.Write(dataReady)
}

func (c *Client) reloadUpdateInfo(mod string) {
	Trace.Printf("[%d. %d] entry into the [%s] mode validation program", c.userIds.Uid, c.userIds.Tid, mod)

	tmpbyte := make([]byte, 4096)

	mapKey := strconv.Itoa(c.userIds.Uid) + "-" + strconv.Itoa(c.userIds.Tid)

	//if _, ok := UPDATEMAP
	file, error := os.OpenFile(".update", os.O_RDONLY, 0766)
	if error != nil {
		fmt.Println(error)
	}
	defer file.Close()

	total, err := file.Read(tmpbyte)
	if err != nil {
		Error.Println(err)
	}

	err = json.Unmarshal(tmpbyte[:total], &UPDATEMAP) // tmpbyte[:total] for error invalid character '\x00' after top-level value
	if err != nil {
		Error.Println(err)
	}

	if _, ok := UPDATEMAP[mapKey]; !ok {
		//new
		Trace.Printf("[%d. %d] is a new connection, exit [%s] mode validation program", c.userIds.Uid, c.userIds.Tid, mod)
	} else if UPDATEMAP[mapKey][6] == "" { // for index out of range UPDATEMAP[mapKey][5] error
		//new
		Trace.Printf("[%d. %d] is a new connection, exit [%s] mode validation program", c.userIds.Uid, c.userIds.Tid, mod)
	} else {
		Trace.Printf("[%d. %d] is a updated before connection, go into [%s] mode program", c.userIds.Uid, c.userIds.Tid, mod)
		//Trace.Println(UPDATEMAP[mapKey])
		c.hub.clients[*c.userIds].Head.rm.RandomName = UPDATEMAP[mapKey][0]
		c.hub.clients[*c.userIds].Head.rm.FtpFileName = UPDATEMAP[mapKey][1]
		c.hub.clients[*c.userIds].Head.sm.Type, _ = strconv.Atoi(UPDATEMAP[mapKey][2])
		c.hub.clients[*c.userIds].Head.rm.Content.ResourceType = UPDATEMAP[mapKey][3]
		c.hub.clients[*c.userIds].Head.ips = UPDATEMAP[mapKey][7]

		//handle selectednodes
		var i int
		i = 0
		for _, v := range strings.Split(UPDATEMAP[mapKey][4], ",") {
			(*(c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes))[i].NodeNames = strings.Split(v, "|")[0]
			(*(c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes))[i].GPUNum, _ = strconv.Atoi(strings.Split(v, "|")[1])
			i++
		}
		statusCode, _ := strconv.Atoi(UPDATEMAP[mapKey][5])

		UPDATEMAP[mapKey][6] = "" // updated, reset to null(need to manually set "" to "1" in .update file)

		// active logs
		if statusCode >= RESOURCECOMPLETE {
			log_back_to_frontend(c, kubeconfigName, nameSpace,
				c.hub.clients[*c.userIds].Head.rm.Content.ResourceType,
				c.hub.clients[*c.userIds].Head.rm.RandomName,
				len(*(c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes)),
				(*(c.hub.clients[*c.userIds].Head.rm.Content.SelectedNodes))[0].GPUNum)
		}
	}
}

/* used to update */
