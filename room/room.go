package room

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Room struct {
	Code                 string
	HostID               string
	Clients              map[*Client]bool
	ClientHistory        map[string]string
	ReconnectTimers      map[string]*time.Timer
	CurrentSong          *spotify.Song
	CurrentSongStartTime time.Time
	Playlist             []spotify.Song
	SkipRecord           []string
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
		r.CurrentSongStartTime = time.Now()
		r.Playlist = r.Playlist[1:]
		r.SkipRecord = r.SkipRecord[:0]

		return nil
	} else {

		event := Event{
			Type:    "last_song_ended",
			Payload: nil,
		}

		for member := range r.Clients {
			member.Egress <- event
		}

		r.SkipRecord = r.SkipRecord[:0]
		r.CurrentSong = nil
		return nil
	}

}

func (r *Room) HandleSkipVote(user string) error {
	r.SkipRecord = append(r.SkipRecord, user)

	payload, err := json.Marshal(VoteToSkipPayload{User: user})
	if err != nil {
		return err
	}

	if len(r.SkipRecord)*2 >= len(r.Clients) {
		err := r.HandleSongChange()
		if err != nil {
			return err
		}

		event := Event{
			Type:    "song_skipped",
			Payload: payload,
		}

		for member := range r.Clients {
			member.Egress <- event
		}
	} else {

		event := Event{
			Type:    "song_skip_vote",
			Payload: payload,
		}

		for member := range r.Clients {
			member.Egress <- event
		}
	}
	return nil
}
