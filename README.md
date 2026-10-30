# Groupie Tracker

A small Go web app for exploring artists, their members, albums, and concert locations. It uses the Groupie Trackers API and renders a responsive artist directory and detail pages.

## Features

- Browse a simple artist directory with circular icons and names; open a profile for the full details.
- Search artists, members, album dates, formation years, and concert locations with type-labeled suggestions.
- Filter by formation year, first-album year, member count, and one or more concert locations.
- Open an artist profile with member information and concert dates grouped by location.
- Load an interactive concert map with geocoded markers.
- Load artist, location, date, and relation data from the API, with a 15-minute in-memory cache.
- Keep serving the last successful catalog if the upstream API is temporarily unavailable.
- Handle invalid routes, IDs, and HTTP methods with clear error pages.

## Requirements

- Go 1.20 or later
- Internet access to load data from the Groupie Trackers API

The application uses only Go standard-library packages.
The interactive map uses Leaflet from its pinned CDN release and OpenStreetMap tiles.

## Run

From the project directory:

```powershell
go run ./cmd
```

Then open [http://localhost:4000](http://localhost:4000).

To use another address or port:

```powershell
go run ./cmd -addr :8080
```

## Data source

Artist information comes from the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api). The app requests the artists, locations, dates, and relation endpoints and combines their records by artist ID.

The map geocodes concert locations through OpenStreetMap Nominatim only after the visitor asks to show the map. Requests are serialized and cached in memory to respect the public service's usage limits. The map shows OpenStreetMap attribution.

You can switch service URLs without changing source code:

```powershell
$env:GROUPIE_API_BASE_URL = 'https://groupietrackers.herokuapp.com/api'
$env:GROUPIE_GEOCODER_URL = 'https://nominatim.openstreetmap.org/search'
$env:GROUPIE_MAP_TILE_URL = 'https://tile.openstreetmap.org/{z}/{x}/{y}.png'
```
