package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"albumart.digital/applemusic"
	"albumart.digital/config"
	"albumart.digital/dotenv"
	"albumart.digital/jsonlog"
)

var (
	ApiPath      string
	ApiPort      string
	ConfigPath   string
	ConfigPort   string
	SupportEmail string
	StatusPage   string
	Port         string
)

func init() {
	dotenv.EnsureSet(
		"API_PATH",
		"CONFIG_PATH",
		"API_PORT",
		"CONFIG_PORT",
	)

	ApiPath = os.Getenv("API_PATH")
	ConfigPath = os.Getenv("CONFIG_PATH")
	SupportEmail = os.Getenv("SUPPORT_EMAIL")
	StatusPage = os.Getenv("STATUS_PAGE")
	ApiPort = os.Getenv("API_PORT")
	ConfigPort = os.Getenv("CONFIG_PORT")

	log.SetFlags(0)
	// log in JSON format to stdout
	log.SetOutput(&jsonlog.Filter{
		Levels: []string{"DEBUG", "INFO", "ERROR"},
		Level:  "INFO",
		Writer: os.Stdout,
	})
}

func main() {
	http.HandleFunc(ApiPath, handleSearch)

	// Config Endpoint
	go http.ListenAndServe(
		fmt.Sprintf(":%s", ConfigPort),
		&config.Handler{},
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%s", ApiPort))
	if err != nil {
		log.Fatalf("ERROR couldn't listen on port %s: %v", Port, err)
	}
	defer l.Close()

	// Start the server
	go func() {
		log.Printf("INFO listening at %v...", ApiPath)
		log.Fatalf("ERROR http.Serve returned with: %v", http.Serve(l, nil))
	}()

	// Handle common process-killing signals so we can gracefully shut down
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigc
	log.Printf("INFO caught signal %s: shutting down.", sig)
}

func ErrorResponse(w http.ResponseWriter, r *http.Request, code int, message string, log_level string, log_msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{message})

	jsonlog.Level(log_level).
		Field("address", r.RemoteAddr).
		Field("status", code).
		Msg(log_msg)
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(struct {
			Message string `json:"message"`
		}{"Only the GET method is supported on this endpoint"})

		jsonlog.Level("INFO").
			Field("address", r.RemoteAddr).
			Field("status", http.StatusMethodNotAllowed).
			Field("method", r.Method).
			Msg("invalid method")
		return
	}

	jsonlog.Level("DEBUG").
		Field("address", r.RemoteAddr).
		Field("query", url.QueryEscape(r.URL.Query().Get("q"))).
		Msg("started processing request")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var err error

	if _, ok := r.URL.Query()["q"]; !ok {
		ErrorResponse(
			w,
			r,
			http.StatusBadRequest,
			"The request was missing the query parameter 'q'",
			"INFO",
			"request missing 'q' parameter",
		)
		return
	}

	albums, err := applemusic.SearchAlbum(r.URL.Query().Get("q"))
	if err != nil {
		ErrorResponse(
			w,
			r,
			http.StatusInternalServerError,
			"The server couldn't complete the request",
			"ERROR",
			fmt.Sprintf("GET request to API failed: %v", err),
		)
		return
	}

	jsonlog.Level("DEBUG").
		Field("parsed-api-response", albums).
		Msg("parsed API response")

	err = json.NewEncoder(w).Encode(albums)
	if err != nil {
		log.Printf("INFO couldn't write response to client: %v", err)
		return
	}

	jsonlog.Level("INFO").
		Field("query", url.QueryEscape(r.URL.Query().Get("q"))).
		Field("status", http.StatusOK).
		Field("address", r.RemoteAddr).
		Field("response-time-ms", time.Since(startTime).Milliseconds()).
		Send()
}
