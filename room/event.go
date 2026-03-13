package room

import (
	"encoding/json"
	"fmt"

	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type EventHandler func(event Event, c *Client) error

type JoinedRoomPayload struct {
	Messsage string   `json:"message"`
	Members  []string `json:"users"`
}

type UserJoinedPayload struct {
	Name string `json:"name"`
}

type SearchSongPayload struct {
	Songs []spotify.Song `json:"songs"`
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

func JoinRoomEvent(event Event, c *Client) error {
	room := c.GetClientRoom()
	var members []string

	for member, ok := range room.Clients {
		if ok {
			members = append(members, member.Name)
		}
	}

	payload := JoinedRoomPayload{
		Messsage: "Joined Room",
		Members:  members,
	}

	marshaledPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	joinedEvent := Event{
		Type:    "joined_room",
		Payload: marshaledPayload,
	}
	c.Egress <- joinedEvent

	userJoinedPayload, err := json.Marshal(UserJoinedPayload{Name: c.Name})
	if err != nil {
		return err
	}

	userJoinedEvent := Event{
		Type:    "user_joined",
		Payload: userJoinedPayload,
	}

	for member, ok := range room.Clients {
		if ok && member != c {
			member.Egress <- userJoinedEvent
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
	room := c.Manager.Rooms[c.RoomCode]

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
	room := c.Manager.Rooms[c.RoomCode]
	err := room.HandleSongChange()
	if err != nil {
		return err
	}

	return nil
}
