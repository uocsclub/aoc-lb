package database

import (
	"log"
	"strconv"
	"strings"

	"uocsclub.net/aoclb/internal/types"
)

func (d *DatabaseInst) GetUserSubmissions(year string, aocUserId int) ([]*types.AOCUserSubmission, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	rows, err := d.db.Query(`SELECT 
			day,
			submission_url,
			language_name,
			modifier_dec_percent
		FROM modifier_submission AS s 
		LEFT JOIN modifiers m ON s.language_name = m.language_name 
		WHERE user_id = ? AND year = ?`, aocUserId, year)
	if err != nil {
		return nil, err
	}

	output := []*types.AOCUserSubmission{}

	for rows.Next() {
		rowData := &types.AOCUserSubmission{}
		var dayString string

		err = rows.Scan(dayString, &rowData.SubmissionUrl, &rowData.LanguageName, &rowData.ModifierDecPercent)
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

func (d *DatabaseInst) GetModifiers() ([]*types.AOCSubmissionModifier, error) {
	d.dbLock.Lock()
	defer d.dbLock.Unlock()

	rows, err := d.db.Query(`SELECT 
			language_name,
			modifier_dec_percent
		FROM modifiers`)
	if err != nil {
		return nil, err
	}

	output := []*types.AOCSubmissionModifier{}

	for rows.Next() {
		rowData := &types.AOCSubmissionModifier{}

		err := rows.Scan(&rowData.LanguageName, &rowData.ModifierDecPercent);
		if err != nil {
			log.Println(err)
			continue
		}

		output = append(output, rowData)
	}

	return output, nil
}
