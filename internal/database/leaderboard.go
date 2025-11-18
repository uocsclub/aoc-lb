package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"uocsclub.net/aoclb/internal/types"
)

func (d *DatabaseInst) GetLeaderboard(year string) (types.AOCData, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	data := types.AOCData{}

	rows, err := d.db.Query("SELECT year, user_id, aoc_user.name, score, day_completions FROM leaderboard_entry LEFT JOIN aoc_user ON aoc_id = user_id WHERE year = ?", year)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		entry := &types.AOCUserLB{
			Completions: map[int]*types.AOCCompletion{},
		}
		var completions string

		err = rows.Scan(&entry.Year, &entry.UserId, &entry.Name, &entry.Score, &completions)

		if err != nil {
			return nil, err
		}

		for completion := range strings.SplitSeq(completions, ",") {
			if len(completion) == 0 {
				continue
			}
			s := strings.Split(completion, "d")

			if len(s) != 2 {
				log.Printf("Got invalid completion format: %s\n", completion)
				continue
			}

			i1, err := strconv.Atoi(s[0])
			if err != nil {
				continue
			}

			if entry.Completions[i1] == nil {
				entry.Completions[i1] = &types.AOCCompletion{}
			}

			if s[1] == "1" {
				entry.Completions[i1].Star1 = true
			}

			if s[1] == "2" {
				entry.Completions[i1].Star2 = true
			}
		}

		data[entry.UserId] = entry
	}

	return data, nil
}

func (d *DatabaseInst) StoreLeaderboard(data types.AOCData) (types.AOCData, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	db, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	err = ensureUsers(db, data)
	if err != nil {
		db.Rollback()
		return nil, err
	}

	for _, entry := range data {
		completions := make([]string, 0, len(entry.Completions)*2)

		for day, completion := range entry.Completions {
			if completion.Star1 {
				completions = append(completions, fmt.Sprintf("%02dd1", day))
			}

			if completion.Star2 {
				completions = append(completions, fmt.Sprintf("%02dd2", day))
			}
		}

		row := db.QueryRow("SELECT user_id FROM leaderboard_entry WHERE year = ? AND user_id = ?", entry.Year, entry.UserId)
		var id int
		if scanErr := row.Scan(&id); scanErr != nil {
			_, err = db.Exec("INSERT INTO leaderboard_entry (year, user_id, score, day_completions) VALUES (?, ?, ?, ?);", entry.Year, entry.UserId, entry.Score, strings.Join(completions, ","))
			if err != nil {
				db.Rollback()
				return nil, err
			}
			continue
		}
		_, err = db.Exec("UPDATE leaderboard_entry SET score = ?, day_completions = ? WHERE year = ? AND user_id = ?;", entry.Score, strings.Join(completions, ","), entry.Year, entry.UserId)
		if err != nil {
			db.Rollback()
			return nil, err
		}
	}

	db.Commit()

	return data, nil
}

func ensureUsers(db *sql.Tx, data types.AOCData) error {
	res, err := db.Query("SELECT aoc_id FROM aoc_user")
	if err != nil {
		return err
	}
	presentUser := map[int]bool{}
	for res.Next() {
		var id int
		err = res.Scan(&id)
		if err != nil {
			return err
		}
		presentUser[id] = true
	}

	for _, user := range data {
		if presentUser[user.UserId] {
			_, err = db.Exec("UPDATE aoc_user SET name=? WHERE aoc_id = ?;", user.Name, user.UserId)
			if err != nil {
				return err
			}
			continue
		}

		_, err = db.Exec("INSERT INTO aoc_user (aoc_id, name) VALUES (?, ?);", user.UserId, user.Name)
		if err != nil {
			return err
		}
	}

	return nil
}
