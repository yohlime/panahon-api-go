package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	missingValue     = 999.9
	minMinutesThresh = -90 * 24 * 60 // 90 days behind
	maxMinutesThresh = 1 * 24 * 60   // 1 day ahead
)

type Lufft struct {
	Obs    StationObservation
	Health StationHealth
}

type StationObservation struct {
	Pres      NullFloat4         `json:"pres"`
	Rr        NullFloat4         `json:"rr"`
	Rh        NullFloat4         `json:"rh"`
	Temp      NullFloat4         `json:"temp"`
	Td        NullFloat4         `json:"td"`
	Wdir      NullFloat4         `json:"wdir"`
	Wspd      NullFloat4         `json:"wspd"`
	Wspdx     NullFloat4         `json:"wspdx"`
	Srad      NullFloat4         `json:"srad"`
	Mslp      NullFloat4         `json:"mslp"`
	Hi        NullFloat4         `json:"hi"`
	Wchill    NullFloat4         `json:"wchill"`
	Timestamp pgtype.Timestamptz `json:"timestamp"`
}

type StationHealth struct {
	Vb1               NullFloat4         `json:"vb1"`
	Vb2               NullFloat4         `json:"vb2"`
	Curr              NullFloat4         `json:"curr"`
	Bp1               NullFloat4         `json:"bp1"`
	Bp2               NullFloat4         `json:"bp2"`
	Cm                NullString         `json:"cm"`
	Ss                NullInt4           `json:"ss"`
	TempArq           NullFloat4         `json:"temp_arq"`
	RhArq             NullFloat4         `json:"rh_arq"`
	Fpm               NullString         `json:"fpm"`
	ErrorMsg          NullString         `json:"error_msg"`
	Message           NullString         `json:"message"`
	DataCount         NullInt4           `json:"data_count"`
	DataStatus        NullString         `json:"data_status"`
	Timestamp         pgtype.Timestamptz `json:"timestamp"`
	MinutesDifference NullInt4           `json:"minutes_difference"`
}

func (l Lufft) String(nVal int) string {
	var wspd, wspdx NullFloat4
	var rr NullInt4
	if l.Obs.Wspd.Valid {
		wspd.Float32 = l.Obs.Wspd.Float32 * 3.6
		wspd.Valid = true
	}
	if l.Obs.Wspdx.Valid {
		wspdx.Float32 = l.Obs.Wspdx.Float32 * 3.6
		wspdx.Valid = true
	}
	if l.Obs.Rr.Valid {
		rr.Int32 = int32(l.Obs.Rr.Float32 / (0.2 * 6.0))
		rr.Valid = true
	}

	obsStr := strings.Join([]string{
		l.Obs.Temp.String(),
		l.Obs.Rh.String(),
		l.Obs.Pres.String(),
		wspd.String(),
		wspdx.String(),
	}, "+")

	if nVal == 20 || nVal == 24 {
		obsStr = obsStr + "+0"
	}

	obsStr = obsStr + "+" + strings.Join([]string{
		l.Obs.Wdir.String(),
		l.Obs.Srad.String(),
		l.Obs.Td.String(),
		l.Obs.Wchill.String(),
		rr.String(),
	}, "+")

	hStr := ""
	if nVal == 23 || nVal == 24 {
		hStr = strings.Join([]string{
			l.Health.Vb1.String(),
			l.Health.Vb2.String(),
			l.Health.Curr.String(),
			l.Health.Bp1.String(),
			l.Health.Bp2.String(),
			l.Health.Cm.String(),
			l.Health.Ss.String(),
			l.Health.TempArq.String(),
			l.Health.RhArq.String(),
			l.Health.Fpm.String(),
		}, "+")
	} else if nVal == 19 {
		hStr = strings.Join([]string{
			l.Health.TempArq.String(),
			l.Health.RhArq.String(),
			l.Health.Ss.String(),
			fmt.Sprintf("%s#", l.Health.Vb1),
			l.Health.Bp1.String(),
			l.Health.Fpm.String(),
		}, "+")
	} else if nVal == 20 {
		hStr = strings.Join([]string{
			l.Health.Ss.String(),
			fmt.Sprintf("%s#", l.Health.Vb1),
			l.Health.Bp1.String(),
			l.Health.TempArq.String(),
			l.Health.RhArq.String(),
			l.Health.Fpm.String(),
		}, "+")
	}

	timestampStr := l.Obs.Timestamp.Time.Format(time.RFC3339)
	if nVal == 23 || nVal == 24 {
		tStr := fmt.Sprintf("%s%s%s/%s%s%s",
			timestampStr[0:4], timestampStr[5:7], timestampStr[8:10],
			timestampStr[11:13], timestampStr[14:16], timestampStr[17:19],
		)
		return fmt.Sprintf("0+%s+0+%s+%s", obsStr, hStr, tStr)
	} else if nVal == 19 || nVal == 20 {
		tStr := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
			timestampStr[0:4], timestampStr[5:7], timestampStr[8:10],
			timestampStr[11:13], timestampStr[14:16], timestampStr[17:19],
		)
		return fmt.Sprintf("0+%s+0+%s+%s", obsStr, hStr, tStr)
	}

	return ""
}

func NewLufftFromString(valStr string) (l *Lufft, err error) {
	valStr = strings.ReplaceAll(valStr, ">", "")
	valStr = strings.ReplaceAll(valStr, "%20", "+")
	valStr = strings.TrimSpace(valStr)

	valStrs := strings.Split(valStr, "+")
	nVal := len(valStrs)

	valStrs = valStrs[1:]

	l = new(Lufft)
	if nVal < 19 {
		return nil, fmt.Errorf("invalid string")
	}

	timeNow := time.Now()
	timestamp := parseTimestampTz(valStrs[nVal-2])
	errMsg := ""
	minutesDiff := 0.0

	if timestamp.Valid {
		minutesDiff = timeNow.Sub(timestamp.Time).Minutes()
		if minutesDiff < minMinutesThresh {
			errMsg = fmt.Sprintf("timestamp is %f minutes behind", math.Abs(minutesDiff))
			timestamp.Time = timeNow
		} else if minutesDiff > maxMinutesThresh {
			errMsg = fmt.Sprintf("timestamp is %f minutes ahead", minutesDiff)
			timestamp.Time = timeNow
		}
	}

	if nVal == 20 || nVal == 24 {
		_v := make([]string, 0)
		_v = append(_v, valStrs[:5]...)
		valStrs = append(_v, valStrs[6:]...)
	}

	if nVal >= 19 {
		l.Obs = StationObservation{
			Temp:      parseFloat(valStrs[0], false),
			Rh:        parseFloat(valStrs[1], false),
			Pres:      parseFloat(valStrs[2], true),
			Wspd:      parseFloatWithCF(valStrs[3], false, 1.0/3.6),
			Wspdx:     parseFloatWithCF(valStrs[4], false, 1.0/3.6),
			Wdir:      parseFloat(valStrs[5], false),
			Srad:      parseFloat(valStrs[6], false),
			Td:        parseFloat(valStrs[7], false),
			Wchill:    parseFloat(valStrs[8], false),
			Rr:        parseFloatWithCF(valStrs[9], false, 0.2*6.0),
			Timestamp: timestamp,
		}
	}

	var health StationHealth
	if nVal == 23 || nVal == 24 {
		health = StationHealth{
			Vb1:  parseFloat(valStrs[11], false),
			Vb2:  parseFloat(valStrs[12], false),
			Curr: parseFloat(valStrs[13], false),
			Bp1:  parseFloat(valStrs[14], false),
			Bp2:  parseFloat(valStrs[15], false),
			Cm: NullString{
				Text: pgtype.Text{
					String: valStrs[16],
					Valid:  true,
				},
			},
			Ss:      parseInt(valStrs[17]),
			TempArq: parseFloat(valStrs[18], false),
			RhArq:   parseFloat(valStrs[19], false),
			Fpm: NullString{
				Text: pgtype.Text{
					String: valStrs[20],
					Valid:  true,
				},
			},
		}
	} else if nVal == 19 {
		health = StationHealth{
			TempArq: parseFloat(valStrs[11], false),
			RhArq:   parseFloat(valStrs[12], false),
			Ss:      parseInt(valStrs[13]),
			Vb1:     parseFloat(strings.Split(valStrs[14], "#")[0], false),
			Bp1:     parseFloat(valStrs[15], false),
			Fpm: NullString{
				Text: pgtype.Text{
					String: valStrs[16],
					Valid:  true,
				},
			},
		}
	} else if nVal == 20 {
		health = StationHealth{
			Ss:      parseInt(valStrs[11]),
			Vb1:     parseFloat((valStrs[12])[:len(valStrs[12])-1], false),
			Bp1:     parseFloat(valStrs[13], false),
			TempArq: parseFloat(valStrs[14], false),
			RhArq:   parseFloat(valStrs[15], false),
			Fpm: NullString{
				Text: pgtype.Text{String: valStrs[16],
					Valid: true},
			},
		}
	}

	dataCount := 0
	dataStatus := ""
	for _, v := range []bool{l.Obs.Temp.Valid, l.Obs.Rh.Valid, l.Obs.Pres.Valid, l.Obs.Wspd.Valid, l.Obs.Wspdx.Valid,
		l.Obs.Wdir.Valid, l.Obs.Srad.Valid, l.Obs.Td.Valid, l.Obs.Wchill.Valid, l.Obs.Rr.Valid,
	} {
		var b2i = map[bool]int8{false: 0, true: 1}
		if v {
			dataCount++
		}
		dataStatus += fmt.Sprintf("%d", b2i[v])
	}

	l.Health = health
	l.Health.Timestamp = timestamp
	l.Health.Message = NullString{
		Text: pgtype.Text{String: valStr,
			Valid: true},
	}
	l.Health.MinutesDifference = NullInt4{
		Int4: pgtype.Int4{
			Int32: int32(minutesDiff),
			Valid: true,
		}}
	l.Health.ErrorMsg = NullString{
		Text: pgtype.Text{String: errMsg,
			Valid: true},
	}
	l.Health.DataCount = NullInt4{
		Int4: pgtype.Int4{
			Int32: int32(dataCount),
			Valid: true,
		},
	}
	l.Health.DataStatus = NullString{
		Text: pgtype.Text{String: dataStatus,
			Valid: true},
	}

	return l, nil
}

func RandomLufft() Lufft {
	timestamp := pgtype.Timestamptz{
		Time:  time.Now(),
		Valid: true,
	}
	return Lufft{
		Obs: StationObservation{
			Temp:      RandomNullFloat4(20, 35),
			Rh:        RandomNullFloat4(0, 100),
			Pres:      RandomNullFloat4(990, 1100),
			Wspd:      RandomNullFloat4(20, 35),
			Wspdx:     RandomNullFloat4(35, 50),
			Wdir:      RandomNullFloat4(0, 359),
			Srad:      RandomNullFloat4(0, 1000),
			Td:        RandomNullFloat4(20, 35),
			Wchill:    RandomNullFloat4(20, 35),
			Rr:        RandomNullFloat4(0, 100),
			Timestamp: timestamp,
		},
		Health: StationHealth{
			Vb1:  RandomNullFloat4(0, 20),
			Vb2:  RandomNullFloat4(0, 20),
			Curr: RandomNullFloat4(0, 1),
			Bp1:  RandomNullFloat4(0, 30),
			Bp2:  RandomNullFloat4(0, 30),
			Cm: NullString{
				Text: pgtype.Text{
					String: RandomString(6),
					Valid:  true,
				},
			},
			Ss:      RandomNullInt4(0, 100),
			TempArq: RandomNullFloat4(20, 35),
			RhArq:   RandomNullFloat4(0, 100),
			Fpm: NullString{Text: pgtype.Text{
				String: RandomString(6),
				Valid:  true,
			}},
			Timestamp: timestamp,
		},
	}
}

func parseFloat(s string, skipValidation bool) NullFloat4 {
	ret := NullFloat4{
		Float4: pgtype.Float4{
			Float32: 0.0,
			Valid:   false,
		},
	}

	val, err := strconv.ParseFloat(s, 32)
	if err != nil || (!skipValidation && val == missingValue) {
		return ret
	}

	ret.Float32 = float32(math.Round(val*100) / 100)
	ret.Valid = true

	return ret
}

func parseFloatWithCF(s string, skipValidation bool, cf float32) NullFloat4 {
	ret := parseFloat(s, skipValidation)

	if ret.Valid {
		ret.Float32 = float32(math.Round(float64(ret.Float32*cf)*100) / 100)
	}

	return ret
}

func parseInt(s string) NullInt4 {
	ret := NullInt4{
		Int4: pgtype.Int4{
			Int32: 0,
			Valid: false,
		},
	}

	val, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return ret
	}

	ret.Int32 = int32(val)
	ret.Valid = true

	return ret
}

func parseTimestampTz(s string) pgtype.Timestamptz {
	var _s string

	ret := pgtype.Timestamptz{
		Valid: false,
	}

	if !strings.Contains(s, "/") {
		_s = fmt.Sprintf("%s-%s-%sT%s:%s:%sZ08:00", s[0:4], s[5:7], s[8:10], s[11:13], s[14:16], s[17:19])
	} else {
		if len(s) == 13 {
			_s = fmt.Sprintf("20%s-%s-%sT%s:%s:%sZ08:00", s[0:2], s[2:4], s[4:6], s[7:9], s[9:11], s[11:13])
		} else {
			_s = fmt.Sprintf("%s-%s-%sT%s:%s:%sZ08:00", s[0:4], s[4:6], s[6:8], s[9:11], s[11:13], s[13:15])
		}
	}

	timestamp, err := time.Parse(time.RFC3339, _s)
	if err != nil {
		return ret
	}

	ret.Time = timestamp

	return ret
}
