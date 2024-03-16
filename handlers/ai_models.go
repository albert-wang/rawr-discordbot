package handlers

type AIModel struct {
	Name     string
	Vision   bool
	Function bool
}

var models = []AIModel{
	GetVisionModel(),
	{"gpt-4-1106-preview", false, true},
	{"gpt-4", false, true},
	{"gpt-3.5-turbo", false, false},
}

func GetVisionModel() AIModel {
	return AIModel{"gpt-4-vision-preview", true, false}
}
