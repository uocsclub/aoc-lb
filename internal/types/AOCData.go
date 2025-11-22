package types

type AOCData = map[int]*AOCUserLB

type AOCUserLB struct {
	Year        string
	User        AOCUser
	Score       int                    // local LB score
	Completions map[int]*AOCCompletion // indexed by day
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
