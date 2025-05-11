package anime

import (
	"fmt"
	"net/url"
	"time"
)

type Status struct {
	Name           string
	CurrentEpisode int64
	LastModified   time.Time
	EpisodeSource  string
	Subgroup       string
}

func (a *Status) FormattedTime() string {
	return a.LastModified.Format("Mon, January 02")
}

func (a *Status) Short() string {
	return fmt.Sprintf("%s - %d (%s)", a.Name, a.CurrentEpisode, a.LastModified.Format("Mon, January 02"))
}

func (a *Status) GetSourceLink() (string, error) {
	if a.EpisodeSource == "" {
		return "", fmt.Errorf("no such anime source")
	}

	parts, err := url.Parse(a.EpisodeSource)
	if err != nil {
		return "", err
	}

	switch parts.Scheme {
	case "cr":
		data, err := ParseCrunchyrollURL(parts)
		if err != nil {
			return "", err
		}

		ep, err := GetBestGuessCrunchyrollLink(data, a.CurrentEpisode)
		if err != nil {
			return "", err
		}

		return ep.Link, nil
	case "nyaa":
		data, err := ParseNyaaURL(parts)
		if err != nil {
			return "", err
		}

		link, err := GetBestGuessNyaaLink(data, a.CurrentEpisode)
		if err != nil {
			return "", err
		}

		return link, nil
	default:
		return "", fmt.Errorf("unknown source scheme: %s", parts.Scheme)
	}
}
