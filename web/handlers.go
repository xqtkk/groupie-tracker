package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w)
		return
	}
	if r.URL.Path != "/" {
		app.NotFound(w)
		return
	}

	artists, err := app.loadCatalog(r.Context())
	if err != nil {
		app.errorLog.Printf("load artist catalog: %v", err)
		app.Errors(w, "Artist data is temporarily unavailable. Please try again shortly.", http.StatusBadGateway)
		return
	}
	if err := app.render(w, "home.html", buildHomePageData(artists)); err != nil {
		app.InternalServerError(w, err)
	}
}

func (app *Application) Artist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		app.MethodNotAllowed(w)
		return
	}
	if r.URL.Path != "/artist" && r.URL.Path != "/artists" {
		app.NotFound(w)
		return
	}
	query := r.URL.Query()
	ids, ok := query["id"]
	if !ok || len(ids) != 1 || len(query) != 1 {
		app.BadRequest(w)
		return
	}
	id, err := strconv.Atoi(ids[0])
	if err != nil || id < 1 || strconv.Itoa(id) != ids[0] {
		app.BadRequest(w)
		return
	}

	artists, err := app.loadCatalog(r.Context())
	if err != nil {
		app.errorLog.Printf("load artist catalog: %v", err)
		app.Errors(w, "Artist data is temporarily unavailable. Please try again shortly.", http.StatusBadGateway)
		return
	}
	for _, artist := range artists {
		if artist.ID == id {
			artist.TileURL = app.Config.TileURL
			if err := app.render(w, "artist.html", artist); err != nil {
				app.InternalServerError(w, err)
			}
			return
		}
	}
	app.BadRequest(w)
}

func (app *Application) ArtistLocations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	ids, ok := r.URL.Query()["id"]
	if r.URL.Path != "/api/artist-locations" || !ok || len(ids) != 1 || len(r.URL.Query()) != 1 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(ids[0])
	if err != nil || id < 1 || strconv.Itoa(id) != ids[0] {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	artists, err := app.loadCatalog(r.Context())
	if err != nil {
		app.errorLog.Printf("load artist catalog for map: %v", err)
		http.Error(w, "Artist data is temporarily unavailable.", http.StatusBadGateway)
		return
	}
	var artist *Artist
	for i := range artists {
		if artists[i].ID == id {
			artist = &artists[i]
			break
		}
	}
	if artist == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	response := struct {
		Points    []LocationPoint `json:"points"`
		Unlocated []string        `json:"unlocated"`
	}{Points: make([]LocationPoint, 0, len(artist.Concerts)), Unlocated: []string{}}
	lookupErrors := 0
	for _, concert := range artist.Concerts {
		coordinates, lookupErr := app.geocoder.Lookup(r.Context(), concert.Location)
		if lookupErr != nil {
			lookupErrors++
			response.Unlocated = append(response.Unlocated, concert.Location)
			continue
		}
		if coordinates == nil {
			response.Unlocated = append(response.Unlocated, concert.Location)
			continue
		}
		response.Points = append(response.Points, LocationPoint{
			Location:  concert.Location,
			Dates:     concert.Dates,
			Latitude:  coordinates.Latitude,
			Longitude: coordinates.Longitude,
		})
	}
	if len(response.Points) == 0 && lookupErrors > 0 {
		http.Error(w, "The map service is temporarily unavailable.", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.errorLog.Printf("encode artist map response: %v", err)
	}
}

// GetResponse is kept as a small compatibility helper for the existing tests.
func (app *Application) GetResponse() ([]ResponseData, error) {
	artists, err := app.loadCatalog(context.Background())
	if err != nil {
		return nil, err
	}
	response := make([]ResponseData, 0, len(artists))
	for _, artist := range artists {
		response = append(response, ResponseData{
			ID:           artist.ID,
			Name:         artist.Name,
			Image:        artist.Image,
			CreationDate: artist.CreationDate,
			FirstAlbum:   artist.FirstAlbum,
			Members:      artist.Members,
		})
	}
	return response, nil
}

func (app *Application) render(w http.ResponseWriter, name string, data any) error {
	var output bytes.Buffer
	if err := templates.ExecuteTemplate(&output, name, data); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := output.WriteTo(w)
	return err
}
