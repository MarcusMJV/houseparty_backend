package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/marcusvorster/houseparty_backend/config"
	"github.com/marcusvorster/houseparty_backend/controllers"
	"github.com/marcusvorster/houseparty_backend/middleware"
	"github.com/marcusvorster/houseparty_backend/room"
)

func main() {
	config.LoadEnv()

	// url, err := config.GenerateSpotifyAuthRequest()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// } else {
	// 	fmt.Println(url)
	// }

	config.InitSpotify()
	RoomManager := room.NewRoomManager()
	controllers.InitRoomManager(RoomManager)

	s, _ := config.GetAccessToken()
	fmt.Println(s)

	server := gin.Default()
	server.Use(middleware.Cors())
	RegisterRoutes(server)
	server.Run(":8080")
}

func RegisterRoutes(server *gin.Engine) {
	server.POST("/create", controllers.CreateRoom)
	server.GET("/join/room/:room_code/:name", controllers.JoinRoom)
	server.GET("/spotify/:code", controllers.SpotifyExchange)
}
