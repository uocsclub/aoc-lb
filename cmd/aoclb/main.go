package main

import (
	"log"
	"os"
	"time"

	"encoding/json"

	"uocsclub.net/aoclb/internal/fetcher"

	"github.com/go-co-op/gocron/v2"
	dotenv "github.com/joho/godotenv"
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

	j, err := s.NewJob(
		gocron.DurationJob(time.Minute/2),
		gocron.NewTask(func() {
			fetcherConfig := fetcher.AOCFetcherConfig{
				SessionCookie: os.Getenv("SESSION_ID"),
				LeaderboardId: os.Getenv("LEADERBOARD_ID"),
				Year:          os.Getenv("YEAR"),
			}

			data, err := fetcher.FetchAOCLeaderboard(&fetcherConfig)

			if err != nil {
				log.Println(err)
			}

			s, _ := json.MarshalIndent(data, "", " ")
			log.Println(string(s))
		},
		),
	)

	s.Start()
	j.RunNow() // durationjob doesn't run on startup

	for true {
	}

}
