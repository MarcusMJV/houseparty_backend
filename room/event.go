package room

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

type JoinedRoomPayload struct {
	Messsage             string            `json:"message"`
	Members              map[string]string `json:"users"`
	Host                 string            `json:"host"`
	ClientID             string            `json:"client_id"`
	CurrentSong          *spotify.Song     `json:"current_song"`
	CurrentSongStartTime time.Time         `json:"current_song_start_time"`
	SkipRecord           []string          `json:"skip_record"`
	Playlist             []spotify.Song    `json:"playlist"`
	Token                string            `json:"token"`
}

type UserJoinedPayload struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type UserLeftPayload struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type SearchSongPayload struct {
	Songs []spotify.Song `json:"songs"`
}

type UpdateStartTimePayload struct {
	StartTime time.Time `json:"start_time"`
}

type VoteToSkipPayload struct {
	User string `json:"user"`
}

type AddToPlaylistPayload struct {
	Song spotify.Song `json:"song"`
}

type SetSongPayload struct {
	Song  spotify.Song `json:"song"`
	Token string       `json:"token"`
}

type AuthTokenPayload struct {
	Token string `json:"token"`
}

type HostUpdatedPayload struct {
	Message              string    `json:"message"`
	Host                 string    `json:"host"`
	Token                string    `json:"token"`
	CurrentSongStartTime time.Time `json:"current_song_start_time"`
}

type SelectedNewHostPayload struct {
	ID string `json:"id"`
}

func JoinRoomEvent(event Event, c *Client) error {
	room := c.GetClientRoom()

	room.mu.Lock()
	defer room.mu.Unlock()

	members := make(map[string]string)

	for member, ok := range room.Clients {
		if ok {
			members[member.Name] = member.ID
		}
	}

	payload := JoinedRoomPayload{
		Messsage:             "Joined Room",
		Members:              members,
		Host:                 room.HostID,
		ClientID:             c.ID,
		CurrentSong:          room.CurrentSong,
		CurrentSongStartTime: room.CurrentSongStartTime,
		SkipRecord:           room.SkipRecord,
		Playlist:             room.Playlist,
	}

	token, err := config.GetAccessToken()
	if err != nil {
		return nil
	}
	payload.Token = token

	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	joinedEvent := Event{
		Type:    "joined_room",
		Payload: marshaledPayload,
	}
	c.Egress <- joinedEvent

	userJoinedPayload, err := json.Marshal(UserJoinedPayload{Name: c.Name, ID: c.ID})
	if err != nil {
		return err
	}

	if _, ok := room.ClientHistory[c.Name]; !ok {
		userJoinedEvent := Event{
			Type:    "user_joined",
			Payload: userJoinedPayload,
		}

		for member, ok := range room.Clients {
			if ok && member != c {
				member.Egress <- userJoinedEvent
			}
		}
	}

	return nil
}

func SearchSongEvent(event Event, c *Client) error {
	var searchPayload struct {
		Search string `json:"search"`
	}

	err := json.Unmarshal(event.Payload, &searchPayload)
	if err != nil {
		return err
	}

	songs, err := spotify.SearchSong(searchPayload.Search)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(SearchSongPayload{
		Songs: songs,
	})
	if err != nil {
		return err
	}

	event = Event{
		Type:    "searched_song",
		Payload: payload,
	}

	c.Egress <- event
	return nil

}

func AddSong(event Event, c *Client) error {
	var song spotify.Song
	room := c.GetClientRoom()

	token, err := config.GetAccessToken()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	err = json.Unmarshal(event.Payload, &song)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	room.mu.Lock()
	defer room.mu.Unlock()

	if room.CurrentSong == nil {
		payload, err := json.Marshal(SetSongPayload{Song: song, Token: token})
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		event = Event{
			Type:    "set_song",
			Payload: payload,
		}

		room.CurrentSong = &song
		room.CurrentSongStartTime = time.Now()

		for member := range room.Clients {
			member.Egress <- event
		}
	} else {
		room.Playlist = append(room.Playlist, song)
		payload, err := json.Marshal(AddToPlaylistPayload{Song: song})
		if err != nil {
			return err
		}

		event = Event{
			Type:    "update_playlist",
			Payload: payload,
		}

		for member := range room.Clients {
			member.Egress <- event
		}
	}

	return nil
}

func SongEnded(event Event, c *Client) error {
	room := c.GetClientRoom()

	room.mu.Lock()
	defer room.mu.Unlock()

	return room.handleSongChange()
}

func VoteToSkipSong(event Event, c *Client) error {
	var skipSongPayload VoteToSkipPayload
	err := json.Unmarshal(event.Payload, &skipSongPayload)
	if err != nil {
		return err
	}

	room := c.GetClientRoom()

	room.mu.Lock()
	defer room.mu.Unlock()

	return room.handleSkipVote(skipSongPayload.User)
}

func SelectedNewHost(event Event, c *Client) error {
	var newHostPayload SelectedNewHostPayload
	err := json.Unmarshal(event.Payload, &newHostPayload)
	if err != nil {
		return err
	}
	room := c.GetClientRoom()

	room.mu.Lock()
	defer room.mu.Unlock()

	newHost := room.getClient(newHostPayload.ID)
	return room.setNewHost(newHost)
}

func UpdateStartTime(event Event, c *Client) error {
	var startTimePayload UpdateStartTimePayload
	err := json.Unmarshal(event.Payload, &startTimePayload)
	if err != nil {
		return err
	}
	room := c.GetClientRoom()

	room.mu.Lock()
	defer room.mu.Unlock()

	room.CurrentSongStartTime = startTimePayload.StartTime
	event.Type = "update_start_time"
	for member := range room.Clients {
		if member.ID != room.HostID {
			member.Egress <- event
		}
	}
	return nil
}
