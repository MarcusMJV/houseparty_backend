package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/marcusvorster/houseparty_backend/config"
)

type Tracks map[string]interface{}

type Song struct {
	Id          string   `json:"id"`
	URI         string   `json:"uri"`
	Name        string   `json:"name"`
	Artists     []string `json:"artists"`
	Album       string   `json:"album"`
	Image       Image    `json:"image"`
	DurationMs  int      `json:"duration_ms"`
	Explicit    bool     `json:"explicit"`
	ExternalURL string   `json:"external_url"`
}

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func SearchSong(search string) ([]Song, error) {
	token, err := config.GetAccessToken()
	if err != nil {
		return nil, err
	}

	query := strings.Replace(search, " ", "+", -1)
	query = strings.Trim(query, " ")

	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://api.spotify.com/v1/search?q="+query+"&type=track&limit=5", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return nil, errors.New("spotify search request failed: " + string(body))
	}

	var responseBody Tracks
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return nil, err
	}

	songs, err := SimplifyTracks(&responseBody)
	if err != nil {
		return nil, err
	}

	return songs, nil
}

func SimplifyTracks(tracks *Tracks) ([]Song, error) {
	var songs []Song

	items := (*tracks)["tracks"].(map[string]interface{})["items"].([]interface{})

	for _, item := range items {
		track := item.(map[string]interface{})

		var artistsNames []string
		if artists, ok := track["artists"].([]interface{}); ok {
			for _, a := range artists {
				artist, ok := a.(map[string]interface{})
				if !ok {
					continue
				}
				artistsNames = append(artistsNames, artist["name"].(string))
			}
		}

		albumData, ok := track["album"].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid album data")
		}
		albumName := albumData["name"].(string)

		var imageURL string
		var imageWidth, imageHeight int
		if images, ok := albumData["images"].([]interface{}); ok && len(images) > 0 {
			image := images[2].(map[string]interface{})
			imageURL = image["url"].(string)
			imageWidth = int(image["width"].(float64))
			imageHeight = int(image["height"].(float64))
		}

		id, _ := track["id"].(string)
		uri, _ := track["uri"].(string)
		name, _ := track["name"].(string)
		durationMs, _ := track["duration_ms"].(float64)
		explicit, _ := track["explicit"].(bool)
		externalURL, _ := track["external_urls"].(map[string]interface{})["spotify"].(string)

		songs = append(songs, Song{
			Id:      id,
			URI:     uri,
			Name:    name,
			Artists: artistsNames,
			Album:   albumName,
			Image: Image{
				URL:    imageURL,
				Width:  imageWidth,
				Height: imageHeight,
			},
			DurationMs:  int(durationMs),
			Explicit:    explicit,
			ExternalURL: externalURL,
		})
	}

	return songs, nil
}
