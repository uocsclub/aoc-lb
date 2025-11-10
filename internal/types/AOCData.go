package types

type AOCData = map[int]*AOCUserLB

type AOCUserLB struct {
	Year        string
	UserId      int
	Score       int // local LB score
	Name        string
	Completions map[int]*AOCCompletion // indexed by day
}

type AOCCompletion struct {
	Star1 bool
	Star2 bool
}
