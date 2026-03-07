package main

import (
	"github.com/gin-gonic/gin"
	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/controllers"
	"github.com/marcusvorster/houseparty_backend/middleware"
	"github.com/marcusvorster/houseparty_backend/room"
)

func main() {
	config.LoadEnv()
	RoomManager := room.NewRoomManager()
	controllers.InitRoomManager(RoomManager)

	server := gin.Default()
	server.Use(middleware.Cors())
	RegisterRoutes(server)
	server.Run(":8080")
}

func RegisterRoutes(server *gin.Engine) {
	server.POST("/create", controllers.CreateRoom)

}
