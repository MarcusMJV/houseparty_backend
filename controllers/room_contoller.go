package controllers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/room"
)

var roomManager *room.RoomManager

func InitRoomManager(m *room.RoomManager) {
	roomManager = m
}

func CreateRoom(context *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}
	if err := context.ShouldBindJSON(&req); err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "error with binding JSON", "error": err.Error()})
		return
	}

	expectedPassword := os.Getenv("DEMO_PASSWORD")

	if req.Password != expectedPassword {
		context.JSON(http.StatusUnauthorized, gin.H{"message": "invalid password", "error": "Invalid Demo Password"})
		return
	}

	code := roomManager.CreateRoom().Code
	context.JSON(http.StatusCreated, gin.H{"message": "room created", "room_code": code})
}

func SpotifyExchange(context *gin.Context) {
	code := context.Param("code")
	token, err := config.SetSpotifyToken(code)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"message": "error getting token", "error": err.Error()})
		return
	}
	context.JSON(http.StatusOK, gin.H{"message": "exchanged", "token": token})
}

func JoinRoom(context *gin.Context) {
	roomManager.ServeWS()(context)
}
