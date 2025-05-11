package anime

import (
	"fmt"
	"log"
	"net/url"
	"os/exec"
	"strconv"
)

type CruncyrollData struct {
	AnimeID      string
	SeasonNumber int64
}

func ParseCrunchyrollURL(source *url.URL) (CruncyrollData, error) {
	season := source.Query().Get("season")
	if season == "" {
		season = "1"
	}

	if source.Opaque == "" {
		return CruncyrollData{}, fmt.Errorf("Invalid Opaque")
	}

	seasonId, err := strconv.ParseInt(season, 10, 64)
	if err != nil {
		return CruncyrollData{}, err
	}

	return CruncyrollData{
		AnimeID:      source.Opaque,
		SeasonNumber: seasonId,
	}, nil
}

type CrunchyrollEpisode struct {
	Link      string
	Thumbnail string
}

func GetBestGuessCrunchyrollLink(cr CruncyrollData, episode int64) (CrunchyrollEpisode, error) {
	// This is dumb but it works without having to mess with avoiding CF
	cmd := exec.Command("./cr_episode", cr.AnimeID, fmt.Sprintf("%d", cr.SeasonNumber), fmt.Sprintf("%d", episode))
	output, err := cmd.Output()
	if err != nil {
		err := err.(*exec.ExitError)
		log.Printf("Couldn't run episode scraper, err=%s stderr=%s", err.Error(), string(err.Stderr))
		return CrunchyrollEpisode{}, err
	}

	return CrunchyrollEpisode{
		Link:      string(output),
		Thumbnail: "",
	}, nil
}
