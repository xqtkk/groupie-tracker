package web

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Application struct {
	errorLog      *log.Logger
	infoLog       *log.Logger
	Config        Config
	httpClient    *http.Client
	geocoder      *Geocoder
	cacheMu       sync.Mutex
	cachedArtists []Artist
	cacheUntil    time.Time
}

type Config struct {
	ArtistsURL  string
	APIBaseURL  string
	GeocoderURL string
	TileURL     string
}

type ApplicationError struct {
	Message string
	Code    int
}

func NewApplication(errorLog, infoLog *log.Logger) *Application {
	apiBaseURL := os.Getenv("GROUPIE_API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "https://groupietrackers.herokuapp.com/api"
	}
	geocoderURL := os.Getenv("GROUPIE_GEOCODER_URL")
	if geocoderURL == "" {
		geocoderURL = "https://nominatim.openstreetmap.org/search"
	}
	tileURL := os.Getenv("GROUPIE_MAP_TILE_URL")
	if tileURL == "" {
		tileURL = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	}
	app := &Application{
		errorLog: errorLog,
		infoLog:  infoLog,
		Config: Config{
			ArtistsURL:  strings.TrimRight(apiBaseURL, "/") + "/artists",
			APIBaseURL:  strings.TrimRight(apiBaseURL, "/"),
			GeocoderURL: geocoderURL,
			TileURL:     tileURL,
		},
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}
	app.geocoder = NewGeocoder(app.Config.GeocoderURL)
	return app
}

func NewServer(addr *string, errorLog *log.Logger, mux *http.ServeMux) *http.Server {
	return &http.Server{
		Addr:         *addr,
		ErrorLog:     errorLog,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

//go:embed templates/*.html
var fs embed.FS

var (
	templates *template.Template
	err       error
)

func init() {
	funcs := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"json": func(value any) string {
			encoded, _ := json.Marshal(value)
			return string(encoded)
		},
		"albumYear": firstAlbumYear,
		"humanize": func(value string) string {
			value = strings.NewReplacer("_", " ", "-", ", ").Replace(value)
			return strings.Title(value)
		},
		"trimDate": func(value string) string { return strings.TrimLeft(value, "*") },
	}
	templates, err = template.New("pages").Funcs(funcs).ParseFS(fs, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
}

func (app *Application) InternalServerError(w http.ResponseWriter, err error) {
	app.errorLog.Println(err)
	app.Errors(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *Application) ClientError(w http.ResponseWriter, status int) {
	app.Errors(w, http.StatusText(status), status)
}

func (app *Application) NotFound(w http.ResponseWriter) {
	app.ClientError(w, http.StatusNotFound)
}

func (app *Application) BadRequest(w http.ResponseWriter) {
	app.ClientError(w, http.StatusBadRequest)
}

func (app *Application) MethodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", http.MethodGet)
	app.ClientError(w, http.StatusMethodNotAllowed)
}

func (app *Application) Errors(w http.ResponseWriter, errorMessage string, errorCode int) {
	buf := new(bytes.Buffer)
	if err := templates.ExecuteTemplate(buf, "error.html", ApplicationError{
		Message: errorMessage,
		Code:    errorCode,
	}); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(errorCode)
	buf.WriteTo(w)
}
