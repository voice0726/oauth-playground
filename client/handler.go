package client

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
	"go.step.sm/crypto/randutil"
	"go.uber.org/zap"
)

var authServer = struct {
	authEndpoint  string
	tokenEndpoint string
}{authEndpoint: "http://localhost:9091/authorize", tokenEndpoint: "http://localhost:9091/token"}

var client = struct {
	redirectURIs []string
	clientID     string
	clientSecret string
}{redirectURIs: []string{"http://localhost:9090/callback"}, clientID: "oauth-client-1", clientSecret: "oauth-client-secret-1"}

type Handler struct {
	httpClient *http.Client
	logger     *zap.Logger
}

func NewHandler(logger *zap.Logger) (*Handler, error) {
	h := &http.Client{}
	return &Handler{httpClient: h, logger: logger}, nil
}

func (h *Handler) HandleIndex(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", nil)
}

func (h *Handler) HandleAuthorize(c echo.Context) error {
	u, _ := url.Parse(authServer.authEndpoint)
	q := u.Query()
	q.Add("response_type", "code")
	q.Add("client_id", client.clientID)
	q.Add("redirect_uri", client.redirectURIs[0])
	q.Add("scope", "")
	state, err := randutil.Alphanumeric(32)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "authorization request failed")
	}
	q.Add("state", state)
	u.RawQuery = q.Encode()
	c.SetCookie(&http.Cookie{Name: "state", Value: state, HttpOnly: true})
	return c.Redirect(http.StatusSeeOther, u.String())
}

func (h *Handler) HandleCallback(c echo.Context) error {
	stateCookie, err := c.Request().Cookie("state")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "failed to parse cookie")
	}
	q := c.Request().URL.Query()
	if stateCookie.Value != q.Get("state") {
		return c.JSON(http.StatusBadRequest, "state not match")
	}

	code := q.Get("code")

	if code != "" {
		return c.JSON(http.StatusBadRequest, "invalid code")
	}

	body := url.Values{}
	body.Add("grant_type", "authorization_code")
	body.Add("code", code)
	body.Add("redirect_uri", client.redirectURIs[0])

	req, err := http.NewRequest("POST", "http://localhost:9091/token", strings.NewReader(body.Encode()))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "failed to create request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Basic "+encodedClientCredentials(client.clientID, client.clientSecret))

	res, err := h.httpClient.Do(req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "authorization request failed")
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "authorization request failed")
	}

	var resBody struct {
		AccessToken string `json:"access_token"`
	}
	h.logger.Debug("got access token", zap.Any("resBody", resBody))
	err = json.Unmarshal(b, &resBody)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "authorization request failed")
	}

	cookie := http.Cookie{Name: "access_token", Value: resBody.AccessToken, HttpOnly: true}
	c.SetCookie(&cookie)
	return c.JSON(http.StatusOK, "ok")
}

func encodedClientCredentials(id, secret string) string {
	return base64.RawStdEncoding.EncodeToString([]byte(id + ":" + secret))
}
