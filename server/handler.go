package server

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"slices"

	"github.com/labstack/echo/v4"
	"github.com/voice0726/oauth-playground/model"
	"github.com/voice0726/oauth-playground/repository"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrClientNotFound error

type Handler struct {
	clientRepository      *repository.ClientRepository
	authRequestRepository *repository.AuthRequestRepository
	logger                *zap.Logger
}

func NewHandler(clientRepo *repository.ClientRepository, authRequestRepository *repository.AuthRequestRepository, logger *zap.Logger) (*Handler, error) {
	return &Handler{clientRepository: clientRepo, authRequestRepository: authRequestRepository, logger: logger}, nil
}

func (h *Handler) HandleIndex(c echo.Context) error {
	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) HandleAuthorize(c echo.Context) error {
	q := c.Request().URL.Query()
	clientID := q.Get("client_id")
	redirectURI := q.Get("redirect_uri")
	resType := q.Get("response_type")
	scope := q.Get("scope")
	state := q.Get("state")

	if clientID == "" || redirectURI == "" || resType == "" {
		return c.Render(http.StatusBadRequest, "error.html", map[string]string{"error": "invalid parameters"})
	}

	client, err := h.getClient(clientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Render(http.StatusBadRequest, "error.html", map[string]string{"error": "invalid client"})
		}
		h.logger.Error("failed to get client: %s", zap.Error(err))
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "failed to get client"})
	}

	if !slices.Contains(client.RedirectURIs, redirectURI) {
		return c.Render(http.StatusBadRequest, "error.html", map[string]string{"error": "invalid redirect uri"})
	}

	req := &model.AuthRequest{
		ClientID:     client.ID,
		RedirectURI:  redirectURI,
		ResponseType: resType,
		State:        state,
		Scope:        scope,
	}

	req, err = h.authRequestRepository.CreateRequest(*req)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "failed to save auth request"})
	}
	// todo: should persist request id here?

	return c.Render(http.StatusOK, "approve.html", map[string]interface{}{"reqid": req.ID.String(), "client": client})
}

func (h *Handler) HandleToken(c echo.Context) error {
	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) HandleApprove(c echo.Context) error {
	var b struct {
		ReqID   string `form:"reqid"`
		Approve string `form:"approve"`
	}
	err := (&echo.DefaultBinder{}).BindBody(c, &b)
	if err != nil {
		h.logger.Error("failed to parse request body", zap.Error(err))
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
	}

	if b.ReqID == "" {
		h.logger.Debug("no reqid provided")
		return c.Render(http.StatusBadRequest, "error.html", map[string]string{"error": "invalid parameters"})
	}

	req, err := h.authRequestRepository.FindRequestByID(b.ReqID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Render(http.StatusBadRequest, "error.html", map[string]string{"error": "invalid request id"})
		}
		h.logger.Error("failed to get request id", zap.Error(err))
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "invalid request id"})
	}

	if b.Approve != "Approve" {
		u, err := url.Parse(req.RedirectURI)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
		}
		q := u.Query()
		q.Add("error", "access_denied")
		u.RawQuery = q.Encode()

		return c.Redirect(http.StatusSeeOther, u.String())
	}

	if req.ResponseType != "code" {
		u, err := url.Parse(req.RedirectURI)
		if err != nil {
			return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
		}
		q := u.Query()
		q.Add("error", "unsupported_response_type")
		u.RawQuery = q.Encode()

		return c.Redirect(http.StatusSeeOther, u.String())
	}

	code := model.AuthCode{
		State: req.State,
	}
	log.Print(code)

	return c.JSON(http.StatusOK, "ok")
}

func (h *Handler) getClient(clientID string) (*model.Client, error) {
	client, err := h.clientRepository.FindClientByName(clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}
