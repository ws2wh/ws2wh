package server

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/net/websocket"
)

type Server struct {
	WsPort     uint
	WhEndpoint string
}

func (s *Server) Serve() {

	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover())
	e.GET("/", handle)
	addr := fmt.Sprintf(":%d", s.WsPort)
	e.Logger.Fatal(e.Start(addr))
}

func handle(c echo.Context) error {
	websocket.Handler(func(ws *websocket.Conn) {
		websocket.Message.Send(ws, "Hello")
		for {
			var msg string
			err := websocket.Message.Receive(ws, &msg)
			if err != nil {
				c.Logger().Error(err)
				break
			}
			c.Logger().Info(msg)
		}
	}).ServeHTTP(c.Response(), c.Request())
	return nil
}
