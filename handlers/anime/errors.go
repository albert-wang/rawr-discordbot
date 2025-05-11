package anime

import (
	"fmt"
)

func UnknownAnime(like string) string {
	return fmt.Sprintf("I don't know anything about %s", like)
}

func NoSource(anime string) string {
	return fmt.Sprintf("No source for %s", anime)
}

func CannotSource(anime string, err error) string {
	return fmt.Sprintf("Couldn't get episode info for anime=%s err=%s", anime, err.Error())
}
