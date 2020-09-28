package main

type Ids struct {
	Uid int `json:"uid"`
	Tid int `json:"tid"`
}

type TrainingData struct {
	Uid              int    `json:"uid"`
	Tid              int    `json:"tid"`
	SelectedModelUrl string `json:"selectedModelUrl"`
	ResourceType     int    `json:"resourceType"`
	SelectedNodes    int    `json:"selectedNodes"`
	ModelType        int    `json:"modelType"`
	//TaskParams []Params `json:"taskParams"`
}

type Params struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
