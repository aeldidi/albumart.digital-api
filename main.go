package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"albumart.digital/auth"
	_ "albumart.digital/dotenv"
)

var (
	//go:embed response.html
	ResponseTemplateFile string
	ResponseTemplate     = template.Must(template.New("Response").Parse(ResponseTemplateFile))
	//go:embed error.html
	ErrorTemplateFile string
	ErrorTemplate     = template.Must(template.New("Error").Parse(ErrorTemplateFile))

	BaseURL      *url.URL
	ApiURL       *url.URL
	AuthClient   *http.Client
	SupportEmail string
	StatusPage   string
	SockPath     string
)

// The JSON response structure.

type Artwork struct {
	Height int64  `json:"height"`
	Width  int64  `json:"width"`
	Url    string `json:"url"`
}

type AlbumAttribute struct {
	ArtistName string  `json:"artistName"`
	Artwork    Artwork `json:"artwork"`
	Name       string  `json:"name"`
}

type Album struct {
	Attributes AlbumAttribute `json:"attributes"`
}

type AlbumSearchResult struct {
	Data []Album `json:"data"`
}

type SearchResults struct {
	Albums AlbumSearchResult `json:"albums"`
}

type APIResponse struct {
	Results SearchResults `json:"results"`
}

func ensureSet(vals ...string) {
	for _, v := range vals {
		if _, found := os.LookupEnv(v); !found {
			log.Fatalf("'%v' not set in .env or environment", v)
		}
	}
}

func init() {
	ensureSet(
		"BASE_URL",
		"APPLE_MUSIC_KEY_ID",
		"APPLE_TEAM_ID",
		"APPLE_PRIVATE_KEY_PATH",
		"SUPPORT_EMAIL",
		"STATUS_PAGE",
		"SOCKET_PATH",
	)

	var err error

	BaseURL, err = url.Parse(os.Getenv("BASE_URL"))
	if err != nil {
		log.Fatalf("couldn't parse base url: %v", err)
	}

	ApiURL, err = url.Parse("https://api.music.apple.com/v1/catalog/us/search")
	if err != nil {
		log.Fatalf("couldn't parse API url: err")
	}

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

	AuthClient = &http.Client{
		Transport: &auth.Transport{
			PrivateKey: pkey,
			KeyId:      os.Getenv("APPLE_MUSIC_KEY_ID"),
			TeamID:     os.Getenv("APPLE_TEAM_ID"),
		},
		Timeout: 5 * time.Second,
	}

	SupportEmail = os.Getenv("SUPPORT_EMAIL")
	StatusPage = os.Getenv("STATUS_PAGE")
	SockPath = os.Getenv("SOCKET_PATH")
}

func main() {
	http.HandleFunc(BaseURL.Path, handleSearch)

	l, err := net.Listen("unix", SockPath)
	if err != nil {
		log.Fatalf("couldn't open '%v': %v", SockPath, err)
	}
	defer l.Close()

	if err := os.Chmod(SockPath, 0666); err != nil {
		log.Fatalf("couldn't set proper permissions for '%v'", SockPath)
	}

	// Start the server
	go func() {
		log.Printf("listening at %v...", BaseURL)
		log.Fatalf("fcgi.Serve returned with: %v", fcgi.Serve(l, nil))
	}()

	// Handle common process-killing signals so we can gracefully shut down:
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	sig := <-sigc
	log.Printf("caught signal %s: shutting down.", sig)
}

func ErrorResponse(w io.Writer) {
	err := struct {
		SupportEmail string
		StatusPage   string
	}{
		SupportEmail: SupportEmail,
		StatusPage:   StatusPage,
	}

	_ = ErrorTemplate.Execute(w, err)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	log.Printf("recieved request from '%v'", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	u := *ApiURL
	q := url.Values{
		"term":  r.URL.Query()["q"],
		"types": []string{"albums"},
		"limit": []string{"25"},
		"with":  []string{"topResults"},
	}
	u.RawQuery = q.Encode()
	resp, err := AuthClient.Get(u.String())
	if err != nil {
		ErrorResponse(w)
		log.Print("couldn't get from the API")
		return
	}

	searchResponse := APIResponse{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ErrorResponse(w)
		log.Printf("couldn't read response body: %v", err)
		return
	}

	err = json.Unmarshal(body, &searchResponse)
	if err != nil {
		ErrorResponse(w)
		log.Print("API returned invalid JSON")
		return
	}

	err = ResponseTemplate.Execute(w, searchResponse.Results.Albums.Data)
	if err != nil {
		if strings.HasSuffix(err.Error(), "use of closed network connection") {
			log.Printf("connection from %v closed", r.RemoteAddr)
			return
		}

		log.Printf("at template: %v", err)
		return
	}
}

func (a Album) Name() string {
	return a.Attributes.Name
}

func (a Album) Small() string {
	smallReplacer := strings.NewReplacer(
		"{w}", "500",
		"{h}", "500",
	)

	return smallReplacer.Replace(a.Attributes.Artwork.Url)
}

func (a Album) Fullsize() string {
	fullsizeReplacer := strings.NewReplacer(
		"{w}", fmt.Sprint(a.Attributes.Artwork.Width),
		"{h}", fmt.Sprint(a.Attributes.Artwork.Height),
	)

	return fullsizeReplacer.Replace(a.Attributes.Artwork.Url)
}

func (a Album) ArtistsName() string {
	return a.Attributes.ArtistName
}
