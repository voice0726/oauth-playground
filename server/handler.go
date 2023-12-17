package server

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/voice0726/oauth-playground/model"
	"github.com/voice0726/oauth-playground/repository"
	"go.step.sm/crypto/randutil"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ErrClientNotFound error

type Handler struct {
	clientRepository      *repository.ClientRepository
	authRequestRepository *repository.AuthRequestRepository
	codeRepostiroy        *repository.CodeRepository
	tokenRepository       *repository.TokenRepository
	logger                *zap.Logger
}

func NewHandler(
	clientRepo *repository.ClientRepository,
	authRequestRepository *repository.AuthRequestRepository,
	codeRepository *repository.CodeRepository,
	tokenRepository *repository.TokenRepository,
	logger *zap.Logger,
) (*Handler, error) {
	return &Handler{clientRepository: clientRepo, authRequestRepository: authRequestRepository, codeRepostiroy: codeRepository, tokenRepository: tokenRepository, logger: logger}, nil
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

	return c.Render(http.StatusOK, "approve.html", map[string]interface{}{"reqid": req.ID.String(), "client": client})
}

func (h *Handler) HandleToken(c echo.Context) error {
	header := c.Request().Header
	auth := header.Get("Authorization")

	var clientID string
	var clientSecret string
	if auth != "" {
		h.logger.Debug("request has an authorization header", zap.ByteString("cred", []byte(strings.TrimPrefix(auth, "Basic "))))
		decoded, err := base64.URLEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "internal server error")
		}
		clientID = strings.Split(string(decoded), ":")[0]
		clientSecret = strings.Split(string(decoded), ":")[1]
	}
	var body struct {
		GrantType    string `form:"grant_type"`
		Code         string `form:"code"`
		ClinetID     string `form:"client_id"`
		ClientSecret string `form:"client_secret"`
		Scope        string `form:"scope"`
	}
	err := c.Bind(&body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}
	h.logger.Debug("incoming request body", zap.Any("body", body))

	if clientID == "" {
		if body.ClinetID == "" {
			h.logger.Info("no clientid provided")
			return c.JSON(http.StatusBadRequest, "client id required")
		}
		clientID = body.ClinetID
	}

	if clientSecret == "" {
		if body.ClientSecret == "" {
			h.logger.Info("no client secret provided")
			return c.JSON(http.StatusBadRequest, "client secret required")
		}
		clientSecret = body.ClientSecret
	}

	if clientID == "" || clientSecret == "" {
		return c.JSON(http.StatusBadRequest, "invalid client")
	}

	client, err := h.clientRepository.FindClientByName(clientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.JSON(http.StatusForbidden, "invalid client ID or credential")
		}
		return c.JSON(http.StatusInternalServerError, "internal server error")
	}

	if client.Secret != clientSecret {
		return c.JSON(http.StatusForbidden, "invalid client ID or credential")
	}

	switch body.GrantType {
	case "authorization_code":
		code, err := h.codeRepostiroy.FindByCode(body.Code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				h.logger.Info("code not found", zap.String("code", body.Code))
				return c.JSON(http.StatusBadRequest, "invalid code")
			}
			return c.JSON(http.StatusInternalServerError, "internal server error")
		}

		// todo: change client name to client id because it's confusing
		if client.Name != clientID {
			h.logger.Info("invalid client id", zap.String("expected", code.ClientID.String()), zap.String("got", clientID))
			return c.JSON(http.StatusBadRequest, "invalid client id")
		}

		token, err := randutil.Alphanumeric(32)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "internal server error")
		}

		t := model.Token{
			Token:    token,
			ClientID: client.ID,
			Scope:    body.Scope,
		}
		log.Print(t)
		_, err = h.tokenRepository.Create(t)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "internal server error")
		}

		return c.JSON(http.StatusOK, map[string]string{
			"access_token": token,
			"token_type":   "Bearer",
			"scope":        body.Scope,
		})

	default:
		h.logger.Info("unknown grant type")
		return c.JSON(http.StatusBadRequest, "unknown grant type")
	}
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

	codeStr, err := randutil.Alphanumeric(8)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
	}
	code := &model.AuthCode{
		Code:     codeStr,
		Scope:    req.Scope,
		ClientID: req.ClientID,
	}
	_, err = h.codeRepostiroy.Create(*code)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
	}

	url, err := url.Parse(req.RedirectURI)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"error": "internal server error"})
	}
	q := url.Query()
	q.Add("code", codeStr)
	q.Add("state", req.State)
	url.RawQuery = q.Encode()

	return c.Redirect(http.StatusSeeOther, url.String())
}

func (h *Handler) getClient(clientID string) (*model.Client, error) {
	client, err := h.clientRepository.FindClientByName(clientID)
	if err != nil {
		return nil, err
	}
	return client, nil
}
