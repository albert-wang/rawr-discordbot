package jikan

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type envelope[T any] struct {
	Data T `json:"data"`
}

type AnimeInformation struct {
	Title         string  `json:"title"`
	URL           string  `json:"url"`
	JapaneseTitle string  `json:"title_japanese"`
	MalID         int     `json:"mal_id"`
	Score         float64 `json:"score"`
	Year          int     `json:"year"`
	Season        string  `json:"season"`
	Status        string  `json:"status"`
	Synopsis      string  `json:"synopsis"`
	Popularity    int     `json:"popularity"`
}

type AnimeDetails struct {
	Title         string `json:"title"`
	JapaneseTitle string `json:"title_japanese"`
	URL           string `json:"url"`
	Synopsis      string `json:"synopsis"`

	Studios []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"studios"`

	Staff      []AnimeStaff     `json:"staff"`
	Characters []AnimeCharacter `json:"characters"`
}

type AnimeFull struct {
	Title         string  `json:"title"`
	JapaneseTitle string  `json:"title_japanese"`
	URL           string  `json:"url"`
	Score         float64 `json:"score"`
	Synopsis      string  `json:"synopsis"`

	Studios []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"studios"`
}

type AnimeStaff struct {
	Person struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"person"`

	Positions []string `json:"positions"`
}

type VoiceActor struct {
	Person struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"person"`

	Language string `json:"language"`
}

type AnimeCharacter struct {
	Character struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"character"`

	Role        string       `json:"role"`
	VoiceActors []VoiceActor `json:"voice_actors"`
}

func apiCall[T any](url string) (T, error) {
	resp, err := http.Get(url)
	if err != nil {
		return *new(T), err
	}

	defer resp.Body.Close()
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return *new(T), err
	}

	if resp.StatusCode != 200 {
		log.Printf("non-200 return code: %d", resp.StatusCode)
		log.Printf("body: %s", string(bytes))
		return *new(T), fmt.Errorf("non-200 return code: %d", resp.StatusCode)
	}

	env := envelope[T]{}
	err = json.Unmarshal(bytes, &env)
	if err != nil {
		return *new(T), err
	}

	return env.Data, nil
}

func GetAnimeDetails(malID int) (AnimeDetails, error) {
	full, err := apiCall[AnimeFull](fmt.Sprintf("https://api.jikan.moe/v4/anime/%d/full", malID))
	if err != nil {
		return AnimeDetails{}, err
	}

	staff, err := apiCall[[]AnimeStaff](fmt.Sprintf("https://api.jikan.moe/v4/anime/%d/staff", malID))
	if err != nil {
		return AnimeDetails{}, err
	}

	if len(staff) > 10 {
		staff = staff[:10]
	}

	characters, err := apiCall[[]AnimeCharacter](fmt.Sprintf("https://api.jikan.moe/v4/anime/%d/characters", malID))
	if err != nil {
		return AnimeDetails{}, err
	}

	filteredCharacters := []AnimeCharacter{}
	for _, character := range characters {
		vas := []VoiceActor{}
		for _, voiceActor := range character.VoiceActors {
			if voiceActor.Language == "Japanese" {
				vas = append(vas, voiceActor)
			}
		}

		character.VoiceActors = vas
		filteredCharacters = append(filteredCharacters, character)
	}

	if len(filteredCharacters) > 10 {
		filteredCharacters = filteredCharacters[:10]
	}

	return AnimeDetails{
		Title:      full.Title,
		URL:        full.URL,
		Studios:    full.Studios,
		Staff:      staff,
		Characters: filteredCharacters,
	}, nil
}

func GetAnime(anime string) ([]AnimeInformation, error) {
	url, err := url.Parse("https://api.jikan.moe/v4/anime")
	if err != nil {
		return nil, err
	}

	q := url.Query()
	q.Set("q", anime)

	url.RawQuery = q.Encode()

	return apiCall[[]AnimeInformation](url.String())
}

func GetSeason(year int, season string) ([]AnimeInformation, error) {
	url, err := url.Parse(fmt.Sprintf("https://api.jikan.moe/v4/seasons/%d/%s", year, season))
	if err != nil {
		return nil, err
	}

	return apiCall[[]AnimeInformation](url.String())
}
