package server

import (
	"html/template"
	"io"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/voice0726/oauth-playground/repository"
	"go.uber.org/zap"
)

type Server struct {
	e      *echo.Echo
	logger *zap.Logger
}

func NewServer(logger *zap.Logger) (*Server, error) {
	e := echo.New()
	e.Renderer = &Template{
		templates: template.Must(template.ParseGlob("server/templates/*.html")),
	}

	dsn := "dev.db"

	clientRepo, err := repository.NewClientRepository(dsn, logger)
	if err != nil {
		return nil, err
	}
	authReqRepo, err := repository.NewAuthRequestRepository(dsn, logger)
	if err != nil {
		return nil, err
	}
	h, err := NewHandler(clientRepo, authReqRepo, logger)

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
	initializeRoutes(e, *h)
	return &Server{e: e, logger: logger}, nil
}

func initializeRoutes(e *echo.Echo, h Handler) {
	e.GET("/", h.HandleIndex)
	e.GET("/authorize", h.HandleAuthorize)
	e.POST("/approve", h.HandleApprove)
	e.GET("/token", h.HandleToken)
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

func (s *Server) Start(address string) error {
	return s.e.Start(address)
}
