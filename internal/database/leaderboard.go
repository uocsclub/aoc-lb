package database

import (
	"database/sql"
	"errors"
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

		err = rows.Scan(&entry.Year, &entry.User.UserId, &entry.User.Name, &entry.Score, &completions)

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

			switch s[1] {
			case "1":
				entry.Completions[i1].Star1 = true
			case "2":
				entry.Completions[i1].Star2 = true
			default:
				log.Printf("Got invalid completion format: %s\n", completion)
				continue
			}

		}
		entry.Modifiers, _ = getUserSubmissionsByFilter(d.db, " user_id = ? AND year = ?", entry.User.UserId, year)

		data[entry.User.UserId] = entry
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

	err = ensureAOCUsers(db, data)
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

		row := db.QueryRow("SELECT user_id FROM leaderboard_entry WHERE year = ? AND user_id = ?", entry.Year, entry.User.UserId)
		var id int
		if scanErr := row.Scan(&id); scanErr != nil {
			_, err = db.Exec("INSERT INTO leaderboard_entry (year, user_id, score, day_completions) VALUES (?, ?, ?, ?);", entry.Year, entry.User.UserId, entry.Score, strings.Join(completions, ","))
			if err != nil {
				db.Rollback()
				return nil, err
			}
			continue
		}
		_, err = db.Exec("UPDATE leaderboard_entry SET score = ?, day_completions = ? WHERE year = ? AND user_id = ?;", entry.Score, strings.Join(completions, ","), entry.Year, entry.User.UserId)
		if err != nil {
			db.Rollback()
			return nil, err
		}
	}

	db.Commit()

	return data, nil
}

func (d *DatabaseInst) GetUserByGithubId(id int) (*types.AOCUser, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	return getUserByGithubId(d.db, id)
}

func getUserByGithubId(db *sql.DB, id int) (*types.AOCUser, error) {
	row := db.QueryRow("SELECT aoc_id, name, github_id, avatar_url FROM aoc_user WHERE github_id = ?;", id)

	user := &types.AOCUser{}
	err := row.Scan(&user.UserId, &user.Name, &user.GithubId, &user.GithubAvatar)
	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return user, nil
}

func (d *DatabaseInst) LinkGithubUser(githubId int, githubAvatar string, aocId int64) (*types.AOCUser, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	user, err := getUserByGithubId(d.db, githubId)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return nil, errors.New("User already paired")
	}

	db, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("UPDATE aoc_user SET github_id = ?, avatar_url = ? WHERE aoc_id = ?", githubId, githubAvatar, aocId)
	if err != nil {
		log.Println(err)
		db.Rollback()
		return nil, err
	}

	db.Commit()

	return getUserByGithubId(d.db, githubId)
}

func ensureAOCUsers(db *sql.Tx, data types.AOCData) error {
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
		if presentUser[user.User.UserId] {
			_, err = db.Exec("UPDATE aoc_user SET name=? WHERE aoc_id = ?;", user.User.Name, user.User.UserId)
			if err != nil {
				return err
			}
			continue
		}

		_, err = db.Exec("INSERT INTO aoc_user (aoc_id, name) VALUES (?, ?);", user.User.UserId, user.User.Name)
		if err != nil {
			return err
		}
	}

	return nil
}
