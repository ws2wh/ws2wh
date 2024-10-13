package server

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"golang.org/x/net/websocket"
)

type Server struct {
	frontendAddr string
	backendUrl   string
}

func Run(frontendAddr string, backendUrl string) Server {
	s := Server{
		frontendAddr: frontendAddr,
		backendUrl:   backendUrl,
	}

	s.serve()
	return s
}

func (s *Server) serve() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover())
	e.GET("/", s.handle)
	e.Logger.Fatal(e.Start(s.frontendAddr))
}

func (s *Server) handle(c echo.Context) error {
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
