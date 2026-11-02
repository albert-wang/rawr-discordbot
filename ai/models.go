package ai

type AIModel struct {
	Name string

	// If this model supports vision
	Vision bool

	// If this model supports function calls
	Function bool
}

var PrimaryModel = AIModel{"gpt-5.4-mini", true, true}

// ThreadModel is the best coding model, used in threads.
var ThreadModel = AIModel{"gpt-5.6-sol", true, true}
