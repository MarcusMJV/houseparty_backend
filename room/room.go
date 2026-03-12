package room

import (
	"math/rand"

	"github.com/marcusvorster/houseparty_backend/spotify"
)

type Room struct {
	Code        string
	HostName    string
	Clients     map[*Client]bool
	CurrentSong spotify.Song
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
