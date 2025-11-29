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
	"time"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/sqlite3/v2"
	"uocsclub.net/aoclb/internal/database"
	"uocsclub.net/aoclb/internal/fetcher"
	"uocsclub.net/aoclb/internal/types"
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
	Year                    string
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
			Expiration: 24 * 7 * time.Hour, // 7 days expiration
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
	s.App.Get("/modifiers", s.HandleModifiers)
	s.App.Get("/about", s.HandleAbout)
	s.App.Get("/usermodifiers", s.HandleUserModifiersGet)
	s.App.Post("/usermodifiers", s.HandleUserModifiersPost)
	s.App.Patch("/usermodifiers", s.HandleUserModifiersPatch)
	s.App.Delete("/usermodifiers", s.HandleUserModifiersDelete)
	s.App.Get("/leaderboard", s.HandleLeaderboard)
	s.App.Get("/", s.HandleRoot)

	s.App.Listen(fmt.Sprintf(":%d", s.config.Port))
	return s
}

func (s *Server) HandleRoot(c *fiber.Ctx) error {
	sess, err := s.store.Get(c)
	if err != nil {
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
	))
}

func (s *Server) HandleLeaderboard(c *fiber.Ctx) error {
	data, err := s.db.GetLeaderboard("2024")
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	return s.Render(c, templates.AOCLeaderboard(data, fetcher.EstimateAOCDayCount(s.config.Year)))
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
	sess.Set("github_id", user.GithubId)

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

	// Fetching gh every api call makes the app feel really slow and unresponsive

	// data, err := FetchGithubOAuthUserEntpoint(token)
	// if err != nil {
	// 	log.Println(err)
	// 	return false
	// }
	//
	// user, err := s.db.GetUserByGithubId(data.GithubUserId)
	// if err != nil || user == nil {
	// 	return false
	// }
	//
	// sess.Set("aoc_id", user.UserId)
	// sess.Set("name", user.Name)
	// sess.Set("github_id", user.GithubId)

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


func (s *Server) HandleAbout(c *fiber.Ctx) error {
	return s.Render(c, templates.About())
}

type userSubmissionFormBody struct {
	SubmissionId  int    `form:"id" query:"id"`
	Day           int    `form:"day"`
	StarId        int    `form:"star"`
	LanguageName  string `form:"language"`
	SubmissionUrl string `form:"submission-url"`
}

func (s *Server) HandleUserModifiersGet(c *fiber.Ctx) error {
	if !s.ValidateGithubLogin(c) {
		return c.SendStatus(http.StatusForbidden)
	}

	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	aocId, ok := sess.Get("aoc_id").(int)
	if !ok {
		return c.SendStatus(http.StatusInternalServerError)
	}

	userSubmissions, err := s.db.GetUserSubmissions(s.config.Year, aocId)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	modifiers, err := s.db.GetModifiers()
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	data := &userSubmissionFormBody{}
	err = c.QueryParser(data)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	var formPrefill *types.AOCUserSubmission = nil
	if data.SubmissionId != 0 {
		submission, err := s.db.GetUserSubmissionById(data.SubmissionId)
		if err == nil && submission != nil && submission.AocUserId == aocId {
			formPrefill = submission
		}
	}

	return s.Render(c, templates.UserModifiers(userSubmissions, modifiers, fetcher.EstimateAOCDayCount(s.config.Year), formPrefill))
}

func (s *Server) HandleUserModifiersPatch(c *fiber.Ctx) error {
	if !s.ValidateGithubLogin(c) {
		return c.SendStatus(http.StatusForbidden)
	}

	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	aocId, ok := sess.Get("aoc_id").(int)
	if !ok {
		return c.SendStatus(http.StatusInternalServerError)
	}

	modifiers, err := s.db.GetModifiers()
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	data := &userSubmissionFormBody{}
	err = c.BodyParser(data)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	submission := &types.AOCUserSubmission{
		AOCSubmissionModifier: types.AOCSubmissionModifier{LanguageName: data.LanguageName},
		AocUserId:             aocId,
		Id:                    data.SubmissionId,
		SubmissionUrl:         data.SubmissionUrl,
		Date:                  data.Day,
		Star:                  data.StarId,
	}

	oldSubmission, err := s.db.GetUserSubmissionById(data.SubmissionId)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}
	if oldSubmission == nil {
		return c.SendStatus(http.StatusNotFound)
	}
	if oldSubmission.AocUserId != aocId {
		return c.SendStatus(http.StatusForbidden)
	}

	if submission.Star < 1 || submission.Star > 2 {
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	dayCount := fetcher.EstimateAOCDayCount(s.config.Year)

	if len(submission.SubmissionUrl) == 0 {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Missing submission url"))
	}

	if submission.Date <= 0 || submission.Date > fetcher.EstimateAOCDayCount(s.config.Year) {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Invalid date"))
	}

	langModifier, err := s.db.GetModifiersByLanguageName(submission.LanguageName)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}
	if langModifier == nil {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Invalid language selection"))
	}

	submission.AOCSubmissionModifier = *langModifier

	newSubmission, err := s.db.UpdateUserSubmission(submission)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "refresh-leaderboard")
	return s.Render(c, templates.OOBUpdateUserModifier(newSubmission, templates.UserModifierForm(modifiers, nil, dayCount, "")))
}

func (s *Server) HandleUserModifiersPost(c *fiber.Ctx) error {
	if !s.ValidateGithubLogin(c) {
		return c.SendStatus(http.StatusForbidden)
	}

	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	aocId, ok := sess.Get("aoc_id").(int)
	if !ok {
		return c.SendStatus(http.StatusInternalServerError)
	}

	modifiers, err := s.db.GetModifiers()
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	data := &userSubmissionFormBody{}
	err = c.BodyParser(data)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	submission := &types.AOCUserSubmission{
		AOCSubmissionModifier: types.AOCSubmissionModifier{LanguageName: data.LanguageName},
		AocUserId:             aocId,
		Id:                    0,
		SubmissionUrl:         data.SubmissionUrl,
		Date:                  data.Day,
		Star:                  data.StarId,
	}

	if submission.Star < 1 || submission.Star > 2 {
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	dayCount := fetcher.EstimateAOCDayCount(s.config.Year)

	if len(submission.SubmissionUrl) == 0 {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Missing submission url"))
	}

	if submission.Date <= 0 || submission.Date > fetcher.EstimateAOCDayCount(s.config.Year) {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Invalid date"))
	}

	langModifier, err := s.db.GetModifiersByLanguageName(submission.LanguageName)
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}
	if langModifier == nil {
		return s.Render(c, templates.UserModifierForm(modifiers, submission, dayCount, "Invalid language selection"))
	}

	submission.AOCSubmissionModifier = *langModifier

	submission, err = s.db.AddUserSubmission(s.config.Year, submission)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "refresh-leaderboard")

	return s.Render(c, templates.OOBAppendUserModifier(submission, templates.UserModifierForm(modifiers, nil, dayCount, "")))
}

func (s *Server) HandleUserModifiersDelete(c *fiber.Ctx) error {
	if !s.ValidateGithubLogin(c) {
		return c.SendStatus(http.StatusForbidden)
	}

	sess, err := s.store.Get(c)
	if err != nil {
		return c.SendStatus(http.StatusInternalServerError)
	}
	aocId, ok := sess.Get("aoc_id").(int)
	if !ok {
		return c.SendStatus(http.StatusInternalServerError)
	}

	data := &userSubmissionFormBody{}
	err = c.QueryParser(data) // delete requests don't have bodies
	if err != nil {
		fmt.Println(err)
		return c.SendStatus(http.StatusUnprocessableEntity)
	}

	submission, err := s.db.GetUserSubmissionById(data.SubmissionId)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}
	if submission == nil {
		return c.SendStatus(http.StatusNotFound)
	}
	if submission.AocUserId != aocId {
		return c.SendStatus(http.StatusForbidden)
	}

	err = s.db.DeleteUserSubmission(data.SubmissionId)
	if err != nil {
		log.Println(err)
		return c.SendStatus(http.StatusInternalServerError)
	}

	c.Set("HX-Trigger", "refresh-leaderboard")
	c.Status(http.StatusOK)
	return c.SendString("")
}
