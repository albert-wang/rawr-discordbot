package ai

import (
	"encoding/json"
	"log"

	"github.com/openai/openai-go/v3/responses"
)

// Tool is a single function the model may call. Tools are registered from
// init() in their own files; ToolDefinitions and InvokeTool are the only
// public entry points the chat loop uses.
type Tool struct {
	Definition responses.FunctionToolParam
	Invoke     func(guild, channel, argsJSON string) []responses.ResponseInputContentUnionParam
}

var registry = map[string]Tool{}

// DefineTool wires a Responses-API function tool definition to a typed
// handler. The generic T is the args struct the handler accepts; JSON
// unmarshal of the model's argument blob happens here so handlers stay
// focused on real logic.
func DefineTool[T any](
	def responses.FunctionToolParam,
	fn func(guild, channel string, args T) []responses.ResponseInputContentUnionParam,
) {
	t := Tool{
		Definition: def,
		Invoke: func(guild, channel, argsJSON string) []responses.ResponseInputContentUnionParam {
			var args T
			if argsJSON != "" {
				if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
					log.Printf("tool %s: bad args %q: %v", def.Name, argsJSON, err)
					return nil
				}
			}
			return fn(guild, channel, args)
		},
	}

	if def.Name == "" {
		panic("ai: tool registered without a name")
	}
	if _, dup := registry[def.Name]; dup {
		panic("ai: duplicate tool registration: " + def.Name)
	}

	registry[def.Name] = t
}

func ToolDefinitions() []responses.ToolUnionParam {
	out := make([]responses.ToolUnionParam, 0, len(registry))
	for _, t := range registry {
		def := t.Definition
		out = append(out, responses.ToolUnionParam{OfFunction: &def})
	}

	out = append(out, responses.ToolUnionParam{
		OfWebSearch: &responses.WebSearchToolParam{
			Type: responses.WebSearchToolTypeWebSearch,
		},
	})
	return out
}

func InvokeTool(guild, channel, name, argsJSON string) []responses.ResponseInputContentUnionParam {
	log.Printf("Invoking tool %s with args %s", name, argsJSON)
	t, ok := registry[name]
	if !ok {
		log.Printf("unknown tool %s", name)
		return nil
	}
	return t.Invoke(guild, channel, argsJSON)
}
