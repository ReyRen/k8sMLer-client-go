package main

// global msg used to recv or send
type msg struct {
	rm        *recvMsg
	sm        *sendMsg
	socketmsg *clientsocketmsg
	cltmp     *Client
}

type recvMsg struct {
	Type        int             `json:"type"`
	Content     *recvMsgContent `json:"content"`
	realPvcName string
}
type recvMsgContent struct {
	IDs              *Ids   `json:"ids"`
	SelectedModelUrl string `json:"selectedModelUrl"`
	ResourceType     string `json:"resourceType"`
	SelectedNodes    int    `json:"selectedNodes"`
	ModelType        int    `json:"modelType"`
	Command          string `json:"command"`
	FrameworkType    int    `json:"frameworkType"`
	Params           string `json:"params"`
	SelectedDataset  string `json:"selectedDataset"`
}
type Ids struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

type sendMsg struct {
	Type    int             `json:"type"`
	Content *sendMsgContent `json:"content"`
}
type sendMsgContent struct {
	Log          string             `json:"log"`
	StatusCode   int                `json:"statusCode"`
	GpuInfo      *sendMsgContentGpu `json:"gpuInfo"`
	ResourceInfo *resourceInfo      `json:"resourceInfo"`
}
type sendMsgContentGpu struct {
	GpuCapacity string `json:"gpuCapacity"`
	GpuUsed     string `json:"gpuUsed"`
}
type resourceInfo struct {
	PodPhase string `json:"podPhase"`
}

// clientsocket send msg
type clientsocketmsg struct {
	Uid      int `json:"uid"`
	Tid      int `json:"tid"`
	StatusId int `json:"statusId"`
}
