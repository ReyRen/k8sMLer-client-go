package main

// global msg used to recv or send
type msg struct {
	rm        *recvMsg
	sm        *sendMsg
	socketmsg *clientsocketmsg
	cltmp     *Client
}

// clientsocket send msg
type clientsocketmsg struct {
	Uid      int    `json:"uid"`
	Tid      int    `json:"tid"`
	StatusId int    `json:"statusId"`
	PodName  string `json:"podName"`
}

//--------------------------------------------------接受消息--------------------------------------------------
type recvMsg struct {
	Type        int             `json:"type"`
	Admin       bool            `json:"admin"`
	Content     *recvMsgContent `json:"content"`
	RandomName  string
	FtpFileName string
}
type recvMsgContent struct {
	IDs                *Ids           `json:"ids"`
	OriginalModelUrl   string         `json:"originalModelUrl"`
	ContinuousModelUrl string         `json:"continuousModelUrl"`
	ModelName          string         `json:"modelName"`
	ResourceType       string         `json:"resourceType"`
	SelectedNodes      *[]selectNodes `json:"selectedNodes"`
	ModelType          int            `json:"modelType"`
	Command            string         `json:"command"`
	FrameworkType      int            `json:"frameworkType"`
	ToolBoxName        string         `json:"toolBoxName"`
	Params             string         `json:"params"`
	SelectedDataset    string         `json:"selectedDataset"`
	ImageName          string         `json:"imageName"`
	DistributingMethod int            `json:"distributingMethod"`
	CommandBox         string         `json:"cmd"`
}
type selectNodes struct {
	NodeNames string `json:"nodeName"`
	GPUNum    int    `json:"gpuNum"`
}
type Ids struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

//--------------------------------------------------发送消息--------------------------------------------------
type sendMsg struct {
	Type    int             `json:"type"`
	Content *sendMsgContent `json:"content"`
}
type sendMsgContent struct {
	Log          string        `json:"log"`
	ResourceInfo *resourceInfo `json:"resourceInfo"`
}
type resourceInfo struct {
	NodesListerName   string `json:"nodesListerName"`
	NodesListerLabel  string `json:"nodesListerLabel"`
	NodesListerStatus string `json:"nodesListerStatus"`
}
