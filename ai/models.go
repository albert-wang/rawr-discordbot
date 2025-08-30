package ai

type AIModel struct {
	Name string

	// If this model supports vision
	Vision bool

	// If this model supports function calls
	Function bool
}

var PrimaryModel = AIModel{"gpt-4o", true, true}
