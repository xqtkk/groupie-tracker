package web

import (
	"net/http"
)

func (app *Application) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/artist", app.Artist)
	mux.HandleFunc("/artists", app.Artist)
	mux.HandleFunc("/api/artist-locations", app.ArtistLocations)

	fileServer := http.FileServer(http.Dir("./ui"))
	mux.Handle("/ui/", http.StripPrefix("/ui", fileServer))

	return mux
}
