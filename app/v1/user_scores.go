package v1

import (
	"fmt"
	"strconv"
	"time"

	"git.zxq.co/ripple/rippleapi/common"
)

type score struct {
	ID         int       `json:"id"`
	BeatmapMD5 string    `json:"beatmap_md5"`
	Score      int64     `json:"score"`
	MaxCombo   int       `json:"max_combo"`
	FullCombo  bool      `json:"full_combo"`
	Mods       int       `json:"mods"`
	Count300   int       `json:"count_300"`
	Count100   int       `json:"count_100"`
	Count50    int       `json:"count_50"`
	CountGeki  int       `json:"count_geki"`
	CountKatu  int       `json:"count_katu"`
	CountMiss  int       `json:"count_miss"`
	Time       time.Time `json:"time"`
	PlayMode   int       `json:"play_mode"`
	Accuracy   float64   `json:"accuracy"`
	PP         float32   `json:"pp"`
}

type userScore struct {
	score
	Beatmap *beatmap `json:"beatmap"`
}

type userScoresResponse struct {
	common.ResponseBase
	Scores []userScore `json:"scores"`
}

const userScoreSelectBase = `
SELECT
	scores.id, scores.beatmap_md5, scores.score,
	scores.max_combo, scores.full_combo, scores.mods,
	scores.300_count, scores.100_count, scores.50_count,
	scores.gekis_count, scores.katus_count, scores.misses_count,
	scores.time, scores.play_mode, scores.accuracy, scores.pp,
	
	beatmaps.beatmap_id, beatmaps.beatmapset_id, beatmaps.beatmap_md5,
	beatmaps.song_name, beatmaps.ar, beatmaps.od, beatmaps.difficulty,
	beatmaps.max_combo, beatmaps.hit_length, beatmaps.ranked,
	beatmaps.ranked_status_freezed, beatmaps.latest_update
FROM scores
LEFT JOIN beatmaps ON beatmaps.beatmap_md5 = scores.beatmap_md5
LEFT JOIN users ON users.username = scores.username
`

// UserScoresBestGET retrieves the best scores of an user, sorted by PP if
// mode is standard and sorted by ranked score otherwise.
func UserScoresBestGET(md common.MethodData) common.CodeMessager {
	cm, wc, param := whereClauseUser(md, "users")
	if cm != nil {
		return *cm
	}
	var modeClause string
	if md.C.Query("mode") != "" {
		m, err := strconv.Atoi(md.C.Query("mode"))
		if err == nil && m >= 0 && m <= 3 {
			modeClause = fmt.Sprintf("AND scores.play_mode = '%d'", m)
		}
	}
	return scoresPuts(md, fmt.Sprintf(
		`WHERE
			scores.completed = '3' 
			AND %s
			%s
			AND users.allowed = '1'
		ORDER BY scores.pp DESC, scores.score DESC %s`,
		wc, modeClause, common.Paginate(md.C.Query("p"), md.C.Query("l"), 100),
	), param)
}

func getMode(m string) string {
	switch m {
	case "1":
		return "taiko"
	case "2":
		return "ctb"
	case "3":
		return "mania"
	default:
		return "std"
	}
}

func scoresPuts(md common.MethodData, whereClause string, params ...interface{}) common.CodeMessager {
	rows, err := md.DB.Query(userScoreSelectBase+whereClause, params...)
	if err != nil {
		md.Err(err)
		return Err500
	}
	var scores []userScore
	for rows.Next() {
		var (
			us userScore
			t  string
			b  beatmapMayOrMayNotExist
		)
		err = rows.Scan(
			&us.ID, &us.BeatmapMD5, &us.Score,
			&us.MaxCombo, &us.FullCombo, &us.Mods,
			&us.Count300, &us.Count100, &us.Count50,
			&us.CountGeki, &us.CountKatu, &us.CountMiss,
			&t, &us.PlayMode, &us.Accuracy, &us.PP,

			&b.BeatmapID, &b.BeatmapsetID, &b.BeatmapMD5,
			&b.SongName, &b.AR, &b.OD, &b.Difficulty,
			&b.MaxCombo, &b.HitLength, &b.Ranked,
			&b.RankedStatusFrozen, &b.LatestUpdate,
		)
		if err != nil {
			md.Err(err)
			return Err500
		}
		// puck feppy
		us.Time, err = time.Parse("060102150405", t)
		if err != nil {
			md.Err(err)
			return Err500
		}
		us.Beatmap = b.toBeatmap()
		scores = append(scores, us)
	}
	r := userScoresResponse{}
	r.Code = 200
	r.Scores = scores
	return r
}