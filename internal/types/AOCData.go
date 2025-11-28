package types

import (
	"fmt"
	"slices"
	"strings"
)

type AOCData = map[int]*AOCUserLB

type AOCUserLB struct {
	Year           string
	User           AOCUser
	Score          int                    // local LB score
	Completions    map[int]*AOCCompletion // indexed by day
	Modifiers      []*AOCUserSubmission
	_adjustedScore int // computed field
}

type AOCUser struct {
	UserId       int
	Name         string
	GithubId     int
	GithubAvatar string
}

type AOCCompletion struct {
	Star1 bool
	Star2 bool
}

type AOCSubmissionModifier struct {
	LanguageName       string
	ModifierDecPercent int // %*10, so 2.5% stored as 25
}

type AOCUserSubmission struct {
	AOCSubmissionModifier
	AocUserId     int
	Id            int
	SubmissionUrl string
	Date          int
	Star          int
}

func SortSubmissionModifiers(modifiers []*AOCSubmissionModifier) {
	slices.SortFunc(modifiers, func(a, b *AOCSubmissionModifier) int {
		diff := b.ModifierDecPercent - a.ModifierDecPercent
		if diff != 0 {
			return diff
		}
		return strings.Compare(a.LanguageName, b.LanguageName)
	})
}

func FormatDecPercent(i int) string {
	return fmt.Sprintf("%d.%d%%", i/10, i%10)
}

func (lb AOCUserLB) GetAdjustedScore() int {
	if lb._adjustedScore != 0 || lb.Score == 0 {
		return lb._adjustedScore
	}
	if len(lb.Modifiers) == 0 {
		return lb.Score
	}
	bestModifiers := map[int][]int{}

	// loop over all modifiers and pick the best ones for the max score
	for _, modifier := range lb.Modifiers {
		if modifier.Star > 2 || modifier.Star < 1 {
			continue
		}
		if bestModifiers[modifier.Date] == nil {
			bestModifiers[modifier.Date] = []int{0, 0}
		}

		bestModifiers[modifier.Date][modifier.Star-1] = max(bestModifiers[modifier.Date][modifier.Star-1], modifier.ModifierDecPercent)
	}

	scoreMultiplier := 1.0
	for _, modifier := range bestModifiers {
		scoreMultiplier += float64(modifier[0])/1000 + float64(modifier[1])/1000
	}

	lb._adjustedScore = int(float64(lb.Score) * scoreMultiplier)

	return lb._adjustedScore
}
