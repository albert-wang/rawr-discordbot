package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
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

func invokeJson(req *http.Request, out interface{}) error {
	b, _ := httputil.DumpRequest(req, true)
	log.Printf("req: %s", string(b))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		b, _ := httputil.DumpRequest(req, true)
		log.Printf("req: %s", string(b))

		b, _ = httputil.DumpResponse(resp, true)
		log.Println(string(b))
		return fmt.Errorf("Non-200 error code from token request: %d", resp.StatusCode)
	}

	return json.Unmarshal(bytes, out)
}

type crTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func getCRToken() (string, error) {
	// Get an auth token
	str := "grant_type=client_id\n"
	tokenReq, err := http.NewRequest("POST", "https://www.crunchyroll.com/auth/v1/token", bytes.NewBuffer([]byte(str)))
	if err != nil {
		return "", err
	}

	tokenReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Add("Accept-Encoding", "gzip, deflate")
	tokenReq.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
	tokenReq.Header.Add("Authorization", fmt.Sprintf("Basic %s", "eHVuaWh2ZWRidDNtYmlzdWhldnQ6MWtJUzVkeVR2akUwX3JxYUEzWWVBaDBiVVhVbXhXMTE="))
	tokenReq.Header.Add("Accept", "application/json, text/plain, */*")

	resp := crTokenResponse{}
	err = invokeJson(tokenReq, &resp)
	if err != nil {
		return "", err
	}

	if resp.AccessToken == "" {
		return "", fmt.Errorf("No access token returned")
	}

	return resp.AccessToken, nil
}

type crSeasonResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Display string `json:"season_display_number"`
		Number  int64  `json:"season_number"`
	} `json:"data"`
}

func getSeasonID(cr CruncyrollData, token string) (string, error) {
	url := fmt.Sprintf("https://www.crunchyroll.com/content/v2/cms/series/%s/seasons?force_locale=&locale=en-US", cr.AnimeID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp := crSeasonResponse{}
	err = invokeJson(req, &resp)
	if err != nil {
		return "", err
	}

	for _, season := range resp.Data {
		if season.Number == cr.SeasonNumber {
			return season.ID, nil
		}
	}

	log.Printf("while looking for season=%d", cr.SeasonNumber)
	log.Print(resp)

	return "", fmt.Errorf("Couldn't find season with number")
}

type CrunchyrollEpisode struct {
	Link      string
	Thumbnail string
}

type crThumbnail struct {
	Height int    `json:"height"`
	Width  int    `json:"width"`
	Source string `json:"source"`
}

func goodThumbnail(thumbs []crThumbnail) *crThumbnail {
	if len(thumbs) == 0 {
		return nil
	}

	best := &thumbs[0]
	for i, thumb := range thumbs {
		if thumb.Width > 1024 && thumb.Width < 1500 {
			best = &thumbs[i]
		}
	}

	return best
}

type crEpisodeResponse struct {
	Data []struct {
		ID        string `json:"id"`
		SlugTitle string `json:"slug_title"`

		Display string `json:"episode"`
		Number  int64  `json:"episode_number"`
		Images  struct {
			Thumbnail []crThumbnail `json:"thumbnail"`
		} `json:"images"`
	} `json:"data"`
}

func getEpisode(season string, targetEpisodeNumber int64, token string) (CrunchyrollEpisode, error) {
	url := fmt.Sprintf("https://www.crunchyroll.com/content/v2/cms/seasons/%s/episodes?locale=en-US", season)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return CrunchyrollEpisode{}, err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp := crEpisodeResponse{}
	err = invokeJson(req, &resp)
	if err != nil {
		return CrunchyrollEpisode{}, err
	}

	for _, episode := range resp.Data {
		if episode.Number == targetEpisodeNumber {
			link := fmt.Sprintf("https://www.crunchyroll.com/watch/%s/%s", episode.ID, episode.SlugTitle)

			res := CrunchyrollEpisode{
				Link:      link,
				Thumbnail: "",
			}

			thumbnail := goodThumbnail(episode.Images.Thumbnail)
			if thumbnail != nil {
				res.Thumbnail = thumbnail.Source
			}

			return res, nil
		}
	}

	log.Printf("while looking for season=%s episode=%d", season, targetEpisodeNumber)
	log.Print(resp)

	return CrunchyrollEpisode{}, fmt.Errorf("Couldn't find season with number")
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
