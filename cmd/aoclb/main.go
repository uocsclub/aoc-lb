package main

import (
	"log"
	"os"
	"time"

	"encoding/json"

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
			return // disable fetching for now

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

			data, err = db.GetLeaderboard(fetcherConfig.Year)
			if err != nil {
				log.Println(err)
			}

			s, _ := json.MarshalIndent(data, "", " ")
			log.Println(string(s))
		},
			db,
		),
	)

	s.Start()
	defer s.Shutdown()
	j.RunNow() // durationjob doesn't run on startup

	web.InitServer(7070, db)

	log.Println("Started!")

	for true {
	}

}
