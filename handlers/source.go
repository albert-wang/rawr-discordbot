package handlers

import (
	"fmt"
	"net/url"
)

func GetSourceLink(anime animeStatus) (string, error) {
	if anime.EpisodeSource == "" {
		return "", fmt.Errorf("No such anime source")
	}

	parts, err := url.Parse(anime.EpisodeSource)
	if err != nil {
		return "", err
	}

	switch parts.Scheme {
	case "cr":
		data, err := ParseCrunchyrollURL(parts)
		if err != nil {
			return "", err
		}

		ep, err := GetBestGuessCrunchyrollLink(data, anime.CurrentEpisode)
		if err != nil {
			return "", err
		}

		return ep.Link, nil
	case "nyaa":
		data, err := ParseNyaaURL(parts)
		if err != nil {
			return "", err
		}

		link, err := GetBestGuessNyaaLink(data, anime.CurrentEpisode)
		if err != nil {
			return "", err
		}

		return link, nil
	default:
		return "", fmt.Errorf("Unknown source scheme: %s", parts.Scheme)
	}
}
