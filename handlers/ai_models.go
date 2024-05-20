package handlers

type AIModel struct {
	Name     string
	Vision   bool
	Function bool
}

var models = []AIModel{
	GetVisionModel(),
}

func GetVisionModel() AIModel {
	return AIModel{"gpt-4o", true, true}
}
