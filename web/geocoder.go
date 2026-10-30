package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Coordinates struct {
	Latitude  float64
	Longitude float64
}

type cachedGeocode struct {
	coordinates Coordinates
	found       bool
}

type Geocoder struct {
	client        *http.Client
	endpoint      string
	mu            sync.Mutex
	nextRequestAt time.Time
	cache         map[string]cachedGeocode
}

func NewGeocoder(endpoint string) *Geocoder {
	if endpoint == "" {
		endpoint = "https://nominatim.openstreetmap.org/search"
	}
	return &Geocoder{
		client:   &http.Client{Timeout: 8 * time.Second},
		endpoint: endpoint,
		cache:    make(map[string]cachedGeocode),
	}
}

func (g *Geocoder) Lookup(ctx context.Context, location string) (*Coordinates, error) {
	key := strings.ToLower(strings.TrimSpace(location))
	if key == "" {
		return nil, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if cached, ok := g.cache[key]; ok {
		if !cached.found {
			return nil, nil
		}
		point := cached.coordinates
		return &point, nil
	}

	if wait := time.Until(g.nextRequestAt); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	g.nextRequestAt = time.Now().Add(time.Second)

	endpoint, err := url.Parse(g.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse geocoder endpoint: %w", err)
	}
	query := endpoint.Query()
	query.Set("q", geocodeQuery(location))
	query.Set("format", "jsonv2")
	query.Set("limit", "1")
	endpoint.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create geocoder request: %w", err)
	}
	req.Header.Set("User-Agent", "GroupieTracker/1.0 (+https://github.com/xqtkk/groupie-tracker)")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("geocode %q: %w", location, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("geocoder returned HTTP %d", resp.StatusCode)
	}
	var results []struct {
		Latitude  string `json:"lat"`
		Longitude string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("decode geocoder response: %w", err)
	}
	if len(results) == 0 {
		g.cache[key] = cachedGeocode{found: false}
		return nil, nil
	}
	latitude, err := strconv.ParseFloat(results[0].Latitude, 64)
	if err != nil || latitude < -90 || latitude > 90 {
		return nil, fmt.Errorf("geocoder returned invalid latitude for %q", location)
	}
	longitude, err := strconv.ParseFloat(results[0].Longitude, 64)
	if err != nil || longitude < -180 || longitude > 180 {
		return nil, fmt.Errorf("geocoder returned invalid longitude for %q", location)
	}
	point := Coordinates{Latitude: latitude, Longitude: longitude}
	g.cache[key] = cachedGeocode{coordinates: point, found: true}
	return &point, nil
}

func geocodeQuery(location string) string {
	location = strings.ReplaceAll(location, "_", " ")
	location = strings.ReplaceAll(location, "-", ", ")
	return strings.TrimSpace(location)
}
