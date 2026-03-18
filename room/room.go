package room

import (
	"encoding/json"
	"math/rand"
	"sync"
	"time"

	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Room struct {
	mu                   sync.RWMutex
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

// handleSongChange is the lock-free inner implementation.
// Callers must hold r.mu before calling.
func (r *Room) handleSongChange() error {
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

func (r *Room) HandleSongChange() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.handleSongChange()
}

// handleSkipVote is the lock-free inner implementation.
// Callers must hold r.mu before calling.
func (r *Room) handleSkipVote(user string) error {
	r.SkipRecord = append(r.SkipRecord, user)

	payload, err := json.Marshal(VoteToSkipPayload{User: user})
	if err != nil {
		return err
	}

	if len(r.SkipRecord)*2 > len(r.Clients) {
		err := r.handleSongChange()
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

func (r *Room) HandleSkipVote(user string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.handleSkipVote(user)
}

// getNewHost is the lock-free inner implementation.
// Callers must hold r.mu before calling.
func (r *Room) getNewHost() *Client {
	var newHost *Client
	for memeber, ok := range r.Clients {
		if ok {
			newHost = memeber
			break
		}
	}
	return newHost
}

func (r *Room) GetNewHost() *Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getNewHost()
}

// setNewHost is the lock-free inner implementation.
// Callers must hold r.mu before calling.
func (r *Room) setNewHost(host *Client) error {
	token, err := config.GetAccessToken()
	if err != nil {
		return err
	}

	r.HostID = host.ID

	payload, err := json.Marshal(HostUpdatedPayload{Message: "You Are Now Host", Host: r.HostID, Token: token, CurrentSongStartTime: r.CurrentSongStartTime})
	if err != nil {
		return err
	}
	event := Event{
		Type:    "set_host",
		Payload: payload,
	}

	host.Egress <- event

	payload, err = json.Marshal(HostUpdatedPayload{Message: "Host Updated", Host: r.HostID, Token: "", CurrentSongStartTime: r.CurrentSongStartTime})
	if err != nil {
		return err
	}
	event = Event{
		Type:    "update_host",
		Payload: payload,
	}

	for member, ok := range r.Clients {
		if member != host {
			if ok {
				member.Egress <- event
			}
		}
	}

	return nil
}

func (r *Room) SetNewHost(host *Client) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.setNewHost(host)
}

// getClient is the lock-free inner implementation.
// Callers must hold r.mu before calling.
func (r *Room) getClient(id string) *Client {
	for member := range r.Clients {
		if member.ID == id {
			return member
		}
	}
	return nil
}

func (r *Room) GetClient(id string) *Client {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getClient(id)
}
