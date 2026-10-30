package web

import (
	"sort"
	"strconv"
	"strings"
)

func buildHomePageData(artists []Artist) HomePageData {
	data := HomePageData{
		Artists:         artists,
		MinCreationYear: 9999,
		MinAlbumYear:    9999,
		MinMemberCount:  9999,
	}
	locationSet := make(map[string]struct{})
	for _, artist := range artists {
		if artist.CreationDate < data.MinCreationYear {
			data.MinCreationYear = artist.CreationDate
		}
		if artist.CreationDate > data.MaxCreationYear {
			data.MaxCreationYear = artist.CreationDate
		}
		if len(artist.Members) < data.MinMemberCount {
			data.MinMemberCount = len(artist.Members)
		}
		if len(artist.Members) > data.MaxMemberCount {
			data.MaxMemberCount = len(artist.Members)
		}
		if year := firstAlbumYear(artist.FirstAlbum); year > 0 {
			if year < data.MinAlbumYear {
				data.MinAlbumYear = year
			}
			if year > data.MaxAlbumYear {
				data.MaxAlbumYear = year
			}
		}
		for _, location := range artist.Locations {
			locationSet[location] = struct{}{}
		}
	}
	if data.MinCreationYear == 9999 {
		data.MinCreationYear = 0
	}
	if data.MinAlbumYear == 9999 {
		data.MinAlbumYear = 0
	}
	if data.MinMemberCount == 9999 {
		data.MinMemberCount = 0
	}
	for location := range locationSet {
		data.Locations = append(data.Locations, location)
	}
	sort.Strings(data.Locations)
	return data
}

func firstAlbumYear(value string) int {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) == 0 {
		return 0
	}
	year, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil || year < 1000 || year > 9999 {
		return 0
	}
	return year
}
