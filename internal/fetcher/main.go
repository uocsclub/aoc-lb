package fetcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"uocsclub.net/aoclb/internal/types"
)

func FetchAOCLeaderboard(config *AOCFetcherConfig) (types.AOCData, error) {

	client := &http.Client{}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://adventofcode.com/%s/leaderboard/private/view/%s.json", config.Year, config.LeaderboardId),
		nil,
	)

	if err != nil {
		log.Println("WARN: Failed to create AOC request body")
		return nil, errors.New("Failed to fetch AOC")
	}

	req.AddCookie(&http.Cookie{
		Name:  "session",
		Value: config.SessionCookie,
		// values after this not required, but this is what AOC uses, and httponly increses security
		Domain:   ".adventofcode.com",
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
	})

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return nil, errors.New("Failed to fetch AOC")
	}

	decoder := json.NewDecoder(resp.Body)

	requestData := AOCResponseLeaderboard{}
	err = decoder.Decode(&requestData)
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return nil, errors.New("Failed to fetch AOC")
	}

	return requestData.ToAOCData(), nil
}
