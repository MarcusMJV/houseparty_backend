package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marcusvorster/houseparty_backend/room"
)

var roomManager *room.RoomManager

func InitRoomManager(m *room.RoomManager) {
	roomManager = m
}

func CreateRoom(context *gin.Context) {
	code := roomManager.CreateRoom().Code
	context.JSON(http.StatusCreated, gin.H{"message": "room created", "room_code": code})
}
