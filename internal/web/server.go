package web

import (
	"fmt"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"uocsclub.net/aoclb/internal/database"
	"uocsclub.net/aoclb/internal/web/templates"
)

type Server struct {
	App *fiber.App
	db  *database.DatabaseInst
}

func InitServer(port int, db *database.DatabaseInst) *Server {
	s := &Server{
		App: fiber.New(),
		db:  db,
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

	s.App.Use("/", s.HandleRoot)

	s.App.Listen(fmt.Sprintf(":%d", port))
	return s
}

func (s *Server) HandleRoot(c *fiber.Ctx) error {
	return s.Render(c, templates.HelloWorld())
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
