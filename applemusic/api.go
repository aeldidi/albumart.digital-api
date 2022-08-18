package applemusic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"albumart.digital/applemusic/auth"
	"albumart.digital/dotenv"
)

// The JSON response structure.

var (
	client *http.Client
	apiURL *url.URL
)

type artwork struct {
	Height int64  `json:"height"`
	Width  int64  `json:"width"`
	Url    string `json:"url"`
}

type albumAttribute struct {
	ArtistName string  `json:"artistName"`
	Artwork    artwork `json:"artwork"`
	Name       string  `json:"name"`
}

type album struct {
	Attributes albumAttribute `json:"attributes"`
}

type albumSearchResult struct {
	Data []album `json:"data"`
}

type searchResults struct {
	Albums albumSearchResult `json:"albums"`
}

type appleAPIResponse struct {
	Results searchResults `json:"results"`
}

func init() {
	dotenv.EnsureSet(
		"APPLE_PRIVATE_KEY_PATH",
		"APPLE_MUSIC_KEY_ID",
		"APPLE_TEAM_ID",
	)

	f, err := os.Open(os.Getenv("APPLE_PRIVATE_KEY_PATH"))
	if err != nil {
		log.Fatalf("couldn't read private key '%v': %v",
			os.Getenv("APPLE_PRIVATE_KEY_PATH"), err)
	}
	defer f.Close()

	pkey, err := auth.LoadPrivateKey(f)
	if err != nil {
		log.Fatalf("couldn't load private key '%v': %v", f.Name(), err)
	}

	client = &http.Client{
		Transport: &auth.Transport{
			PrivateKey: pkey,
			KeyId:      os.Getenv("APPLE_MUSIC_KEY_ID"),
			TeamID:     os.Getenv("APPLE_TEAM_ID"),
		},
		Timeout: 5 * time.Second,
	}

	apiURL, err = url.Parse("https://api.music.apple.com/v1/catalog/us/search")
	if err != nil {
		log.Fatalf("couldn't parse API url: err")
	}
}

type Album struct {
	Name         string `json:"name"`
	ArtistName   string `json:"artist-name"`
	SmallArtwork string `json:"artwork-small"`
	Artwork      string `json:"artwork"`
}

func SearchAlbum(searchterm string) ([]Album, error) {
	u := *apiURL
	q := url.Values{
		"term":  []string{searchterm},
		"types": []string{"albums"},
		"limit": []string{"25"},
		"with":  []string{"topResults"},
	}
	u.RawQuery = q.Encode()
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf(
			"making API request for search term '%v': %w",
			searchterm,
			err,
		)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"reading API response for search term '%v': %w",
			searchterm,
			err,
		)
	}

	searchResponse := appleAPIResponse{}
	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		return nil, fmt.Errorf(
			"unmarshaling JSON response for search term '%v': %w",
			searchterm,
			err,
		)
	}

	result := make([]Album, len(searchResponse.Results.Albums.Data))
	for idx, album := range searchResponse.Results.Albums.Data {
		attr := album.Attributes
		replacer := strings.NewReplacer(
			"{w}", fmt.Sprint(attr.Artwork.Width),
			"{h}", fmt.Sprint(attr.Artwork.Height),
		)
		smallReplacer := strings.NewReplacer(
			"{w}", "500",
			"{h}", "500",
		)

		result[idx] = Album{
			Name:         attr.Name,
			ArtistName:   attr.ArtistName,
			Artwork:      replacer.Replace(attr.Artwork.Url),
			SmallArtwork: smallReplacer.Replace(attr.Artwork.Url),
		}
	}

	return result, nil
}
