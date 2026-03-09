package room

import "encoding/json"

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
