package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type apiArtist struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Members      []string `json:"members"`
}

type apiIndex[T any] struct {
	Index []T `json:"index"`
}

type apiLocations struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
}

type apiDates struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

type apiRelation struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

type Concert struct {
	Location string
	Dates    []string
}

type Artist struct {
	ID           int
	Name         string
	Image        string
	CreationDate int
	FirstAlbum   string
	Members      []string
	Locations    []string
	Dates        []string
	Concerts     []Concert
	TileURL      string
}

type ResponseData struct {
	ID           int
	Name         string
	Image        string
	CreationDate int
	FirstAlbum   string
	Members      []string
}

type HomePageData struct {
	Artists         []Artist
	Locations       []string
	MinCreationYear int
	MaxCreationYear int
	MinAlbumYear    int
	MaxAlbumYear    int
	MinMemberCount  int
	MaxMemberCount  int
}

type LocationPoint struct {
	Location  string   `json:"location"`
	Dates     []string `json:"dates"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
}

func (app *Application) loadCatalog(ctx context.Context) ([]Artist, error) {
	app.cacheMu.Lock()
	defer app.cacheMu.Unlock()

	if time.Now().Before(app.cacheUntil) && len(app.cachedArtists) > 0 {
		return app.cachedArtists, nil
	}

	base := strings.TrimRight(app.Config.APIBaseURL, "/")
	if base == "" {
		base = "https://groupietrackers.herokuapp.com/api"
	}

	var artists []apiArtist
	var locations apiIndex[apiLocations]
	var dates apiIndex[apiDates]
	var relations apiIndex[apiRelation]
	err := fetchAPISet(ctx, app.httpClient, map[string]any{
		base + "/artists":   &artists,
		base + "/locations": &locations,
		base + "/dates":     &dates,
		base + "/relation":  &relations,
	})
	if err != nil {
		// Keep the last known good catalog available if the upstream API is
		// temporarily unavailable.
		if len(app.cachedArtists) > 0 {
			app.cacheUntil = time.Now().Add(time.Minute)
			return app.cachedArtists, nil
		}
		return nil, err
	}

	locationByID := make(map[int][]string, len(locations.Index))
	for _, item := range locations.Index {
		locationByID[item.ID] = item.Locations
	}
	datesByID := make(map[int][]string, len(dates.Index))
	for _, item := range dates.Index {
		datesByID[item.ID] = item.Dates
	}
	relationsByID := make(map[int]map[string][]string, len(relations.Index))
	for _, item := range relations.Index {
		relationsByID[item.ID] = item.DatesLocations
	}

	catalog := make([]Artist, 0, len(artists))
	for _, item := range artists {
		artist := Artist{
			ID:           item.ID,
			Name:         item.Name,
			Image:        item.Image,
			CreationDate: item.CreationDate,
			FirstAlbum:   item.FirstAlbum,
			Members:      item.Members,
			Locations:    locationByID[item.ID],
			Dates:        datesByID[item.ID],
		}

		relation := relationsByID[item.ID]
		for location, schedule := range relation {
			dates := append([]string(nil), schedule...)
			artist.Concerts = append(artist.Concerts, Concert{Location: location, Dates: dates})
		}
		if len(artist.Concerts) == 0 && len(artist.Locations) > 0 {
			for i, location := range artist.Locations {
				var schedule []string
				if i < len(artist.Dates) {
					schedule = []string{artist.Dates[i]}
				}
				artist.Concerts = append(artist.Concerts, Concert{Location: location, Dates: schedule})
			}
		}
		sort.Slice(artist.Concerts, func(i, j int) bool {
			return artist.Concerts[i].Location < artist.Concerts[j].Location
		})
		catalog = append(catalog, artist)
	}

	app.cachedArtists = catalog
	app.cacheUntil = time.Now().Add(15 * time.Minute)
	return catalog, nil
}

func fetchAPISet(ctx context.Context, client *http.Client, requests map[string]any) error {
	type result struct{ err error }
	results := make(chan result, len(requests))
	for url, destination := range requests {
		go func(url string, destination any) {
			results <- result{err: fetchJSON(ctx, client, url, destination)}
		}(url, destination)
	}
	var firstErr error
	for range requests {
		if res := <-results; res.err != nil && firstErr == nil {
			firstErr = res.err
		}
	}
	return firstErr
}

func fetchJSON(ctx context.Context, client *http.Client, url string, destination any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create API request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request API: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API returned HTTP %d", resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(destination); err != nil {
		return fmt.Errorf("decode API response: %w", err)
	}
	return nil
}
