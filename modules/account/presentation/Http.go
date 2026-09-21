package presentation

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mehdihadeli/go-mediatr"

	"UnpakSiamida/common/helper"
	commoninfra "UnpakSiamida/common/infrastructure"
	commonpresentation "UnpakSiamida/common/presentation"
	login "UnpakSiamida/modules/account/application/Login"
	who "UnpakSiamida/modules/account/application/Whoami"
	domainaccount "UnpakSiamida/modules/account/domain"
)

// =======================================================
// POST /account
// =======================================================

// LoginHandler godoc
// @Summary Create new Account
// @Tags Account
// @Param username formData string true "Username"
// @Param password formData string true "Password"
// @Produce json
// @Success 200 {object} map[string]string "jwt"
// @Failure 400 {object} commoninfra.ResponseError
// @Failure 404 {object} commoninfra.ResponseError
// @Failure 409 {object} commoninfra.ResponseError
// @Failure 500 {object} commoninfra.ResponseError
// @Router /account [post]
func LoginHandlerfunc(c *fiber.Ctx) error {
	username := c.FormValue("username")
	password := c.FormValue("password")

	cmd := login.LoginCommand{
		Username: username,
		Password: password,
	}

	result, err := mediatr.Send[login.LoginCommand, *domainaccount.LoginResult](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(fiber.Map{
		// "user_id":       result.UserID,
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
	})
}

func WhoAmIHandler(c *fiber.Ctx) error {
	userID := c.FormValue("sid")

	cmd := who.WhoamiCommand{
		SID: userID,
	}
	result, err := mediatr.Send[who.WhoamiCommand, *domainaccount.Account](context.Background(), cmd)
	if err != nil {
		return commoninfra.HandleError(c, err)
	}

	return c.JSON(result)
}

func AvatarProxyHandler(c *fiber.Ctx) error {
	email := c.Query("email")
	username := c.Query("username")
	name := c.Query("name")
	refresh := c.Query("refresh")

	baseURL := os.Getenv("AVATAR_API_URL")
	if baseURL == "" {
		baseURL = "http://172.16.45.91/api/avatar"
	}

	reqURL, err := url.Parse(baseURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Invalid avatar base URL")
	}

	q := reqURL.Query()
	if email != "" {
		q.Set("email", email)
	}
	if username != "" {
		q.Set("username", username)
	}
	if name != "" {
		q.Set("name", name)
	}
	if refresh == "1" {
		q.Set("refresh", "1")
	}
	reqURL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", reqURL.String(), nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to build avatar request")
	}
	req.Header.Set("Host", "api.id.unpak.ac.id")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[Avatar] Upstream fetch error: %v", err)
		initial := "U"
		if len(name) > 0 {
			initial = strings.ToUpper(string([]rune(name)[0]))
		}
		c.Set("Content-Type", "image/svg+xml")
		return c.SendString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" rx="50" fill="#1e293b"/><text x="50%%" y="54%%" font-family="system-ui, -apple-system, sans-serif" font-weight="900" font-size="44" fill="#ffffff" dominant-baseline="middle" text-anchor="middle">%s</text></svg>`, initial))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to read avatar response")
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=900, stale-while-revalidate=86400")
	if src := resp.Header.Get("X-Avatar-Source"); src != "" {
		c.Set("X-Avatar-Source", src)
	}
	if cacheStatus := resp.Header.Get("X-Cache-Status"); cacheStatus != "" {
		c.Set("X-Cache-Status", cacheStatus)
	}

	return c.Status(resp.StatusCode).Send(body)
}

func ModuleAccount(app *fiber.App) {
	pubKey, err := helper.LoadRSAPublicKey("public.pem")

	if err != nil {
		log.Fatal(err)
	}

	app.Post("/api/login", LoginHandlerfunc)
	app.Get("/api/whoami", commonpresentation.SmartCompress(), commonpresentation.JWTMiddleware(pubKey), WhoAmIHandler)
	app.Get("/api/avatar", AvatarProxyHandler)
}
