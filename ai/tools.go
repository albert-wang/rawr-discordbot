package ai

import (
	"encoding/json"
	"log"

	openai "github.com/sashabaranov/go-openai"
)

// Tool is a single function the model may call. Tools are registered from
// init() in their own files; ToolDefinitions and InvokeTool are the only
// public entry points the chat loop uses.
type Tool struct {
	Definition openai.Tool
	Invoke     func(guild, channel, argsJSON string) []openai.ChatMessagePart
}

var registry = map[string]Tool{}

// DefineTool wires an OpenAI function definition to a typed handler. The
// generic T is the args struct the handler accepts; JSON unmarshal of the
// model's argument blob happens here so handlers stay focused on real logic.
func DefineTool[T any](
	def openai.FunctionDefinition,
	fn func(guild, channel string, args T) []openai.ChatMessagePart,
) {
	t := Tool{
		Definition: openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &def,
		},
		Invoke: func(guild, channel, argsJSON string) []openai.ChatMessagePart {
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

	if t.Definition.Function == nil {
		panic("ai: tool registered without a function definition")
	}
	name := t.Definition.Function.Name
	if _, dup := registry[name]; dup {
		panic("ai: duplicate tool registration: " + name)
	}

	registry[name] = t
}

func ToolDefinitions() []openai.Tool {
	out := make([]openai.Tool, 0, len(registry))
	for _, t := range registry {
		out = append(out, t.Definition)
	}
	return out
}

func InvokeTool(guild, channel, name, argsJSON string) []openai.ChatMessagePart {
	log.Printf("Invoking tool %s with args %s", name, argsJSON)
	t, ok := registry[name]
	if !ok {
		log.Printf("unknown tool %s", name)
		return nil
	}
	return t.Invoke(guild, channel, argsJSON)
}
