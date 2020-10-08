package main

type Ids struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

type recvMsg struct {
	Type        int            `json:"type"`
	Content     recvMsgContent `json:"content"`
	realPvcName string
	/*Uid              int    `json:"uid"`
	Tid              int    `json:"tid"`
	SelectedModelUrl string `json:"selectedModelUrl"`
	ResourceType     string `json:"resourceType"`
	SelectedNodes    int    `json:"selectedNodes"`
	ModelType        int    `json:"modelType"`
	Command          string `json:"command"`
	realPvcName      string*/
	//TaskParams recvMsgParams `json:"taskParams"`
}
type recvMsgContent struct {
	Uid              int    `json:"uid"`
	Tid              int    `json:"tid"`
	SelectedModelUrl string `json:"selectedModelUrl"`
	ResourceType     string `json:"resourceType"`
	SelectedNodes    int    `json:"selectedNodes"`
	ModelType        int    `json:"modelType"`
	Command          string `json:"command"`
}

/*type recvMsgParams struct {
	Label string `json:"label"`
	Value string `json:"value"`
}*/

type sendMsg struct {
	Type    int            `json:"type"`
	Content sendMsgContent `json:"content"`
}
type sendMsgContent struct {
	Log        string            `json:"log"`
	StatusCode int               `json:"statusCode"`
	GpuInfo    sendMsgContentGpu `json:"gpuInfo"`
}
type sendMsgContentGpu struct {
	GpuCapacity string `json:"gpuCapacity"`
	GpuUsed     string `json:"gpuUsed"`
}
