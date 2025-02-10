package handlers

type AIModel struct {
	Name string

	// If this model supports vision
	Vision bool

	// If this model supports function calls
	Function bool
}

var models = []AIModel{
	GetVisionModel(),
}

func GetVisionModel() AIModel {
	return AIModel{"gpt-4o", true, true}
}
