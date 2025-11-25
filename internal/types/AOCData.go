package types

type AOCData = map[int]*AOCUserLB

type AOCUserLB struct {
	Year        string
	User        AOCUser
	Score       int                    // local LB score
	Completions map[int]*AOCCompletion // indexed by day
	Modifiers   []*AOCUserSubmission
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
	LanguageName    string
	ModifierDecPercent int // %*10, so 2.5% stored as 25
}

type AOCUserSubmission struct {
	AOCSubmissionModifier
	SubmissionUrl   string
	Date            int
	Star            int
}
