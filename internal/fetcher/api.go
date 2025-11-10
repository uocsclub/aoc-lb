package fetcher

import "uocsclub.net/aoclb/internal/types"

type AOCResponseLeaderboard struct {
	OwnerId        int                              `json:"owner_id"`
	NumDays        int                              `json:"num_days"`
	StartTimestamp int                              `json:"day1_ts"`
	Year           string                           `json:"event"`
	Members        map[string]*AOCLeaderboardMember `json:"members,omitempty"`
}

type AOCLeaderboardMember struct {
	Id                int                                  `json:"id"`
	LocalScore        int                                  `json:"local_score"`
	Name              string                               `json:"name"`
	LastStarTimestamp int                                  `json:"last_star_ts"`
	GlobalScore       int                                  `json:"global_score"` // global leaderboard deleted in 2025 iirc
	Stars             int                                  `json:"stars"`
	DayCompletions    map[int]*AOCLeaderboardDayCompletion `json:"completion_day_level,omitempty"`
}

type AOCLeaderboardDayCompletion struct {
	Star1 *AOCLeaderboardStarCompletion `json:"1,omitempty"`
	Star2 *AOCLeaderboardStarCompletion `json:"2,omitempty"`
}

type AOCLeaderboardStarCompletion struct {
	Index  int `json:"star_index"`
	StarTS int `json:"get_star_Ts"`
}

func (l *AOCResponseLeaderboard) ToAOCData() types.AOCData {
	if l == nil {
		return nil
	}

	data := types.AOCData{}

	for _, member := range l.Members {
		entry := &types.AOCUserLB{
			UserId:      member.Id,
			Score:       member.LocalScore,
			Name:        member.Name,
			Year:        l.Year,
			Completions: map[int]*types.AOCCompletion{},
		}

		for id, day := range member.DayCompletions {
			entry.Completions[id] = &types.AOCCompletion{
				Star1: day.Star1 != nil,
				Star2: day.Star2 != nil,
			}
		}

		data[member.Id] = entry
	}

	return data
}
