package main

// global msg used to recv or send
type msg struct {
	rm    *recvMsg
	sm    *sendMsg
	cltmp *Client
}

type recvMsg struct {
	Type        int             `json:"type"`
	Content     *recvMsgContent `json:"content"`
	realPvcName string
}
type recvMsgContent struct {
	IDs *Ids `json:"ids"`
	/*Uid              int    `json:"uid"`
	Tid              int    `json:"tid"`*/
	SelectedModelUrl string `json:"selectedModelUrl"`
	ResourceType     string `json:"resourceType"`
	SelectedNodes    int    `json:"selectedNodes"`
	ModelType        int    `json:"modelType"`
	Command          string `json:"command"`
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
	Log        string             `json:"log"`
	StatusCode int                `json:"statusCode"`
	GpuInfo    *sendMsgContentGpu `json:"gpuInfo"`
}
type sendMsgContentGpu struct {
	GpuCapacity string `json:"gpuCapacity"`
	GpuUsed     string `json:"gpuUsed"`
}
