package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type NyaaData struct {
	SearchString string
}

// nyaa://search-string
func ParseNyaaURL(source *url.URL) (NyaaData, error) {
	if source.Opaque == "" {
		return NyaaData{}, fmt.Errorf("No valid nyaa data?")
	}

	return NyaaData{
		SearchString: source.Opaque,
	}, nil
}

type titleScoreParts struct {
	Is1080                    bool
	IsHVEC                    bool
	IsH265                    bool
	Is10Bit                   bool
	ContainsEpisodeNumber     bool
	ContainsEnglishOrMultiple bool
}

type nyaaLink struct {
	Href  string
	Title string

	Likes int64
	score titleScoreParts
}

func (t *nyaaLink) Score() int64 {
	accumulatedScore := t.Likes

	if t.score.Is1080 {
		accumulatedScore += 100000
	}

	if t.score.IsHVEC {
		accumulatedScore += 100000
	}

	if t.score.IsH265 {
		accumulatedScore += 100000
	}

	if t.score.Is10Bit {
		accumulatedScore += 100000
	}

	if t.score.ContainsEpisodeNumber {
		accumulatedScore += 500000
	}

	if t.score.ContainsEnglishOrMultiple {
		accumulatedScore += 1000000
	}

	return accumulatedScore
}

func GetBestGuessNyaaLink(data NyaaData, episode int64) (string, error) {
	base, err := url.Parse("https://nyaa.si")
	if err != nil {
		return "", err
	}

	query := url.Values{}
	query.Add("f", "0")
	query.Add("c", "0_0")
	query.Add("q", fmt.Sprintf("%s %d", data.SearchString, episode))

	base.RawQuery = query.Encode()
	log.Print("getting ", base.String())
	resp, err := http.Get(base.String())
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Print(err)
		return "", err
	}

	candidates := []nyaaLink{}
	doc.Find(".torrent-list tbody tr").Each(func(i int, sel *goquery.Selection) {
		sel.Find("td").Each(func(tdIndex int, sel2 *goquery.Selection) {
			cols, exists := sel2.Attr("colspan")
			if exists && cols == "2" {
				sel2.Find("a:last-child").Each(func(j int, link *goquery.Selection) {
					candidates = append(candidates, nyaaLink{
						Href:  link.AttrOr("href", ""),
						Title: strings.TrimSpace(link.Text()),
					})
				})
			}

			// Probably likes?
			if tdIndex == 7 {
				likes, err := strconv.ParseInt(strings.TrimSpace(sel2.Text()), 10, 64)
				if err == nil {
					if len(candidates) > 0 {
						candidates[len(candidates)-1].Likes = likes
					}
				}
			}
		})
	})

	paddedEpisodeNumberWithSpace := fmt.Sprintf("%02d ", episode)

	for i, c := range candidates {
		lowerTitle := strings.ToLower(c.Title)

		candidates[i].score = titleScoreParts{
			Is1080:                    strings.Contains(lowerTitle, "1080p"),
			IsHVEC:                    strings.Contains(lowerTitle, "hvec"),
			IsH265:                    strings.Contains(lowerTitle, "h265"),
			Is10Bit:                   strings.Contains(lowerTitle, "10bit"),
			ContainsEpisodeNumber:     strings.Contains(lowerTitle, paddedEpisodeNumberWithSpace),
			ContainsEnglishOrMultiple: strings.Contains(lowerTitle, "eng") || strings.Contains(lowerTitle, "multi"),
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("No candidates in %s", base.String())
	}

	slices.SortFunc(candidates, func(a nyaaLink, b nyaaLink) int {
		return int(b.Score() - a.Score())
	})

	best := candidates[0]
	return fmt.Sprintf("%s (or %s if that one is wrong)",
		fmt.Sprintf("https://nyaa.si/%s", best.Href),
		base.String()), nil
}
