package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

var (
	//go:embed all:views
	viewsFS embed.FS

	//go:embed all:public
	publicFS embed.FS

	//go:embed db/migrations/*.sql
	migrationsFS embed.FS

	// fiber setting
	port = getEnv("PORT", "8080")

	// database settings
	dbHost     = getEnv("DB_HOST", "localhost")
	dbName     = getEnv("DB_NAME", "postgres")
	dbUser     = getEnv("DB_USER", "demo")
	dbPassword = getEnv("DB_PASSWORD", "demo")
	dbPort     = getEnv("DB_PORT", "5432")
)

type DadJoke struct {
	Setup     string
	Punchline string
}

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	if err = migrateDatabase(db); err != nil {
		log.Fatal(err)
	}

	engine := html.NewFileSystem(http.FS(viewsFS), ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())
	app.Use("/public", filesystem.New(filesystem.Config{
		Root:       http.FS(publicFS),
		PathPrefix: "public",
		Browse:     true,
		MaxAge:     86400,
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		var joke DadJoke

		row := db.QueryRow(`SELECT setup, punchline FROM dad_jokes ORDER by random() LIMIT 1`)
		if err := row.Scan(&joke.Setup, &joke.Punchline); err != nil {
			log.Println(err)
			joke = DadJoke{
				Setup:     "What is a child guilty of if they refuse to nap?",
				Punchline: "Resisting a rest",
			}
		}

		return c.Render("views/index", fiber.Map{
			"DadJoke": joke,
		})
	})

	log.Fatal(app.Listen(fmt.Sprintf(":%s", port)))
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func migrateDatabase(db *sql.DB) error {
	d, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		log.Fatal(err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", d, dbName, driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}

		log.Fatal(err)
	}

	return nil
}
