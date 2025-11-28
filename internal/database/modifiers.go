package database

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"uocsclub.net/aoclb/internal/types"
)

func (d *DatabaseInst) GetUserSubmissions(year string, aocUserId int) ([]*types.AOCUserSubmission, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	return getUserSubmissionsByFilter(d.db, "user_id = ? AND year = ?", aocUserId, year)
}

func (d *DatabaseInst) AddUserSubmission(year string, submission *types.AOCUserSubmission) (*types.AOCUserSubmission, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	db, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	row := db.QueryRow(`
		INSERT INTO modifier_submission (
		year,
		user_id,
		day,
		submission_url,
		language_name) 
		VALUES (?, ?, ?, ?, ?) RETURNING id;
		`,
		year,
		submission.AocUserId,
		fmt.Sprintf("%02dd%d", submission.Date, submission.Star),
		submission.SubmissionUrl,
		submission.LanguageName,
	)

	var id int
	if scanErr := row.Scan(&id); scanErr != nil {
		db.Rollback()
		log.Println(scanErr)
		return nil, scanErr
	}

	db.Commit()

	submission.Id = id
	return submission, nil
}

func (d *DatabaseInst) UpdateUserSubmission(submission *types.AOCUserSubmission) (*types.AOCUserSubmission, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	db, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		UPDATE modifier_submission SET
		day = ?,
		submission_url = ?,
		language_name = ?
		WHERE id = ?;
		`,
		fmt.Sprintf("%02dd%d", submission.Date, submission.Star),
		submission.SubmissionUrl,
		submission.LanguageName,
		submission.Id,
	)

	if err != nil {
		db.Rollback()
		log.Println(err)
		return nil, err
	}

	db.Commit()

	return submission, nil
}

func (d *DatabaseInst) DeleteUserSubmission(submissionId int) error {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	db, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DELETE FROM modifier_submission WHERE id = ?; `,
		submissionId,
	)

	if err != nil {
		db.Rollback()
		log.Println(err)
		return err
	}

	db.Commit()

	return nil
}

func (d *DatabaseInst) GetUserSubmissionById(submissionId int) (*types.AOCUserSubmission, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	submissions, err := getUserSubmissionsByFilter(d.db, " id = ?", submissionId)
	if err != nil {
		return nil, err
	}
	if len(submissions) == 0 {
		return nil, nil
	}

	return submissions[0], nil
}

func (d *DatabaseInst) GetModifiers() ([]*types.AOCSubmissionModifier, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	return getModifiersByFilter(d.db, "")
}

func (d *DatabaseInst) GetModifiersByLanguageName(languageName string) (*types.AOCSubmissionModifier, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	modifiers, err := getModifiersByFilter(d.db, " language_name = ? ", languageName)
	if err != nil {
		return nil, err
	}
	if len(modifiers) == 0 {
		return nil, nil
	}

	return modifiers[0], nil
}

func getUserSubmissionsByFilter(db *sql.DB, filter string, args ...any) ([]*types.AOCUserSubmission, error) {

	query := `SELECT 
			day,
			submission_url,
			m.language_name,
			modifier_dec_percent,
			user_id,
			id
		FROM modifier_submission AS s 
		LEFT JOIN modifiers m ON s.language_name = m.language_name`
	if len(filter) != 0 {
		query += " WHERE " + filter
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	output := []*types.AOCUserSubmission{}

	for rows.Next() {
		rowData := &types.AOCUserSubmission{}
		var dayString string

		err = rows.Scan(&dayString, &rowData.SubmissionUrl, &rowData.LanguageName, &rowData.ModifierDecPercent, &rowData.AocUserId, &rowData.Id)
		if err != nil {
			log.Println(err)
			continue
		}

		if len(dayString) == 0 {
			continue
		}
		s := strings.Split(dayString, "d")

		if len(s) != 2 {
			log.Printf("Got invalid day format: %s\n", dayString)
			continue
		}

		i1, err := strconv.Atoi(s[0])
		if err != nil {
			continue
		}
		rowData.Date = i1

		switch s[1] {
		case "1":
			rowData.Star = 1
		case "2":
			rowData.Star = 2
		default:
			log.Printf("Got invalid day format: %s\n", dayString)
			continue
		}

		output = append(output, rowData)
	}

	return output, nil
}

func getModifiersByFilter(db *sql.DB, filter string, args ...any) ([]*types.AOCSubmissionModifier, error) {

	query := `SELECT 
			language_name,
			modifier_dec_percent
		FROM modifiers`

	if len(filter) != 0 {
		query += " WHERE " + filter
	}
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	output := []*types.AOCSubmissionModifier{}

	for rows.Next() {
		rowData := &types.AOCSubmissionModifier{}

		err := rows.Scan(&rowData.LanguageName, &rowData.ModifierDecPercent)
		if err != nil {
			log.Println(err)
			continue
		}

		output = append(output, rowData)
	}

	return output, nil
}
