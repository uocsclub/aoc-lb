package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"uocsclub.net/aoclb/internal/fetcher"
	"uocsclub.net/aoclb/internal/web"

	"github.com/go-co-op/gocron/v2"
	dotenv "github.com/joho/godotenv"
	"uocsclub.net/aoclb/internal/database"
)

func main() {
	err := dotenv.Load()
	if err != nil {
		log.Println("WARN: Failed to load .env")
	}

	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalln("Failed to start scheduler")
	}

	db, err := database.InitDatabase("./data.sqlite3", "./migrations")
	if err != nil {
		log.Println(err)
		return
	}

	j, err := s.NewJob(
		gocron.DurationJob(time.Minute/2),
		gocron.NewTask(func(db *database.DatabaseInst) {
			// return // disable fetching for now

			fetcherConfig := fetcher.AOCFetcherConfig{
				SessionCookie: os.Getenv("SESSION_ID"),
				LeaderboardId: os.Getenv("LEADERBOARD_ID"),
				Year:          os.Getenv("YEAR"),
			}

			data, err := fetcher.FetchAOCLeaderboard(&fetcherConfig)

			if err != nil {
				log.Println(err)
			}

			_, err = db.StoreLeaderboard(data)
			if err != nil {
				log.Println(err)
			}
		},
			db,
		),
	)

	s.Start()
	defer s.Shutdown()
	j.RunNow() // durationjob doesn't run on startup

	port := os.Getenv("SERVER_PORT")
	if len(port) == 0 {
		port = "7071"
	}
	iport, err := strconv.Atoi(port)
	if err != nil {
		log.Println("Failed to parse SERVER_PORT env variable")
	}

	web.InitServer(web.ServerConfig{
		Port:                    iport,
		OAuth2GithubClientId:    os.Getenv("GITHUB_OAUTH_ID"),
		OAuth2GithubRedirectURI: os.Getenv("GITHUB_OAUTH_REDIRECT_URI"),
		OAuth2GithubSecret:      os.Getenv("GITHUB_OAUTH_SECRET"),
	}, db)

	log.Println("Started!")

	for true {
	}

}
