package client

import (
	"html/template"
	"io"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

type Server struct {
	e  *echo.Echo
	lg *zap.Logger
}

func NewServer(logger *zap.Logger) (*Server, error) {
	e := echo.New()
	e.Renderer = &Template{
		templates: template.Must(template.ParseGlob("client/templates/*.html")),
	}

	h, err := NewHandler(logger)
	if err != nil {
		return nil, err
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogMethod: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			logger.Info("request",
				zap.String("URI", v.URI),
				zap.String("method", v.Method),
				zap.Int("status", v.Status),
			)

			return nil
		},
	}))

	initRoute(e, *h)
	return &Server{e: e, lg: logger}, nil
}

func initRoute(e *echo.Echo, h Handler) {
	e.GET("/", h.HandleIndex)
	e.GET("/authorize", h.HandleAuthorize)
	e.GET("/callback", h.HandleCallback)
}

func (s *Server) Start(address string) error {
	return s.e.Start(address)
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	err := t.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		log.Print(err)
	}
	return err
}
