package controllers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

func JoinRoom(context *gin.Context) {
	roomManager.ServeWS()(context)
}
