package room

import (
	"encoding/json"
	"math/rand"

	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Room struct {
	Code        string
	HostName    string
	Clients     map[*Client]bool
	CurrentSong *spotify.Song
	Playlist    []spotify.Song
}

const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRoomCode(length int) string {
	code := make([]byte, length)

	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}

	return string(code)
}

func (r *Room) HandleSongChange() error {
	if len(r.Playlist) > 0 {
		token, err := config.GetAccessToken()
		if err != nil {
			return err
		}

		payload, err := json.Marshal(AuthTokenPayload{Token: token})
		if err != nil {
			return err
		}
		event := Event{
			Type:    "play_next",
			Payload: payload,
		}

		for member := range r.Clients {
			member.Egress <- event
		}

		r.CurrentSong = &r.Playlist[0]
		r.Playlist = r.Playlist[1:]

		return nil
	} else {

		event := Event{
			Type:    "last_song",
			Payload: nil,
		}

		for member := range r.Clients {
			member.Egress <- event
		}

		r.CurrentSong = nil
		return nil
	}
}
