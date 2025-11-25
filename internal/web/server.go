package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	"uocsclub.net/aoclb/internal/database"
	"uocsclub.net/aoclb/internal/web/templates"
)

type Server struct {
	App    *fiber.App
	db     *database.DatabaseInst
	config ServerConfig
	store  *session.Store
}

type ServerConfig struct {
	Port                    int
	OAuth2GithubClientId    string
	OAuth2GithubRedirectURI string
	OAuth2GithubSecret      string
}

func InitServer(config ServerConfig, db *database.DatabaseInst) *Server {
	s := &Server{
		App:    fiber.New(),
		db:     db,
		config: config,
		store: session.New(session.Config{
			Storage: sqlite3.New(sqlite3.Config{
				Database: "./fiber_storage.sqlite3",
			}),
		}),
	}

	s.App.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS,PATCH",
		AllowHeaders:     "Accept,Authorization,Content-Type",
		AllowCredentials: false, // credentials require explicit origins
		MaxAge:           300,
	}))

	s.App.Use(func(c *fiber.Ctx) error {
		// Ensures that the htmx headers are present
		return c.Next()
	})

	s.App.Use("/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(AssetsEFS),
		PathPrefix: "assets",
		Browse:     false,
	}))

	s.App.Get("/oauth2", s.HandleOAuthRedir)
	s.App.Post("/oauth2", s.HandleOauthLink)
	s.App.Get("/logout", s.HandleLogout)
	s.App.Use("/modifiers", s.HandleModifiers)
	s.App.Use("/", s.HandleRoot)

	s.App.Listen(fmt.Sprintf(":%d", s.config.Port))
	return s
}

func (s *Server) HandleRoot(c *fiber.Ctx) error {
	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	data, err := s.db.GetLeaderboard("2024")
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	redirectUri, err := url.JoinPath(s.config.OAuth2GithubRedirectURI, "/oauth2")
	if err != nil {
		log.Printf("Redirect URI invalid: %s\n", redirectUri)
	}

	var loginWidget templ.Component
	loggedIn := false

	if s.ValidateGithubLogin(c) {
		loggedIn = true
		name, ok := sess.Get("name").(string)
		if ok {
			loginWidget = templates.LoggedInWidget(name)
			goto logged_out
		}
	}
	loginWidget = templates.GithubLogin(s.config.OAuth2GithubClientId, redirectUri)
logged_out:

	return s.Render(c, templates.LandingPage(
		loginWidget,
		loggedIn,
		data,
		25,
	))
}

func (s *Server) HandleOAuthRedir(c *fiber.Ctx) error {
	oauth2Code := c.Query("code", "")
	if len(oauth2Code) == 0 {
		redirect(c, "/")
	}

	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	defer sess.Save()

	client := &http.Client{}

	body := url.Values{}
	body.Add("client_id", s.config.OAuth2GithubClientId)
	body.Add("client_secret", s.config.OAuth2GithubSecret)
	body.Add("code", oauth2Code)

	req, err := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		strings.NewReader(body.Encode()),
	)
	req.Header.Add("Accept", "application/json")

	if err != nil {
		log.Println("WARN: Failed to create github oauth2 request")
		return c.SendStatus(http.StatusInternalServerError)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: Failed to fetch github access_token %s\n", err)
		return c.SendStatus(http.StatusInternalServerError)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return redirect(c, "/")
	}

	parsedBody := struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
		TokenType   string `json:"token_type"`
	}{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&parsedBody)
	if err != nil {
		return redirect(c, "/")
	}

	if len(parsedBody.AccessToken) == 0 {
		return redirect(c, "/")
	}

	sess.Set("access_token", parsedBody.AccessToken)

	data, err := FetchGithubOAuthUserEntpoint(parsedBody.AccessToken)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	user, err := s.db.GetUserByGithubId(data.GithubUserId)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	if user == nil {
		return s.Render(c, templates.OAuthReturn(false))
	}

	sess.Set("aoc_id", user.UserId)
	sess.Set("name", user.Name)

	return redirect(c, "/")
}

func (s *Server) HandleOauthLink(c *fiber.Ctx) error {
	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	defer sess.Save()

	token, ok := sess.Get("access_token").(string)
	if !ok || len(token) == 0 {
		fmt.Println("Invalid access token")
		return redirect(c, "/")
	}

	aocId_s := c.FormValue("aoc-id", "")
	if len(aocId_s) == 0 {
		return s.Render(c, templates.OAuthReturn(true))
	}
	fmt.Println(aocId_s)
	aocId_s = strings.TrimPrefix(aocId_s, "#")
	aocId, err := strconv.ParseInt(aocId_s, 10, 64)
	if err != nil {
		return s.Render(c, templates.OAuthReturn(true))
	}

	data, err := FetchGithubOAuthUserEntpoint(token)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	user, err := s.db.LinkGithubUser(data.GithubUserId, data.AvatarUrl, aocId)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	if user == nil {
		return s.Render(c, templates.OAuthReturn(true))
	}

	sess.Set("aoc_id", user.UserId)
	sess.Set("name", user.Name)

	return redirect(c, "/")
}

func (s *Server) Render(c *fiber.Ctx, component templ.Component) error {
	c.Set("Content-Type", "text/html")
	context := c.Context()

	renderOrder := []func(templ.Component) templ.Component{}

	if c.Get("HX-Request") != "true" {
		renderOrder = append(renderOrder, templates.Index)
	}

	// we need to render bottom-up
	for i := len(renderOrder) - 1; i >= 0; i -= 1 {
		component = renderOrder[i](component)
	}

	return component.Render(context, c.Response().BodyWriter())
}

func redirect(c *fiber.Ctx, target string) error {
	// if there is htmx loaded, force a full redirect
	if c.Get("HX-Request") == "true" {
		c.Set("HX-Redirect", target)
		return c.SendStatus(200)
	}

	// no HTMX, native redirect will work
	return c.Redirect(target)
}

type GithubUserEndpointData struct {
	AvatarUrl    string `json:"avatar_url"`
	GithubUserId int    `json:"id"`
}

func FetchGithubOAuthUserEntpoint(accessToken string) (*GithubUserEndpointData, error) {
	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		"https://api.github.com/user",
		nil,
	)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Add("X-GitHub-Api-Version", "2022-11-28")

	if err != nil {
		return nil, errors.New("WARN: Failed to create github api request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("ERROR: Failed to fetch github user endpoint %s\n", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("ERROR: Failed to fetch github user endpoint, status: %d\n", resp.StatusCode))
	}

	parsedBody := GithubUserEndpointData{}

	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&parsedBody)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("ERROR: Failed to parse user endpoint body")
	}

	return &parsedBody, nil
}

func (s *Server) ValidateGithubLogin(c *fiber.Ctx) bool {
	sess, err := s.store.Get(c)
	if err != nil {
		return false
	}
	defer sess.Save()

	token, ok := sess.Get("access_token").(string)
	if !ok || len(token) == 0 {
		return false
	}

	data, err := FetchGithubOAuthUserEntpoint(token)
	if err != nil {
		log.Println(err)
		return false
	}

	user, err := s.db.GetUserByGithubId(data.GithubUserId)
	if err != nil || user == nil {
		return false
	}

	sess.Set("aoc_id", user.UserId)
	sess.Set("name", user.Name)

	return true
}

func (s *Server) HandleLogout(c *fiber.Ctx) error {
	sess, err := s.store.Get(c)
	if err == nil {
		sess.Destroy()
	}

	return redirect(c, "/")
}

func (s *Server) HandleModifiers(c *fiber.Ctx) error {
	modifiers, err := s.db.GetModifiers()
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	return s.Render(c, templates.ModifiersPage(modifiers))
}
