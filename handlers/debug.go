package handlers

import (
	"encoding/json"
	"log"
)

func dbg(something any) {
	b, _ := json.MarshalIndent(something, "", "  ")
	log.Print(string(b))
}
