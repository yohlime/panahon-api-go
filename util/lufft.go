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
	Pres      pgtype.Float4      `json:"pres"`
	Rr        pgtype.Float4      `json:"rr"`
	Rh        pgtype.Float4      `json:"rh"`
	Temp      pgtype.Float4      `json:"temp"`
	Td        pgtype.Float4      `json:"td"`
	Wdir      pgtype.Float4      `json:"wdir"`
	Wspd      pgtype.Float4      `json:"wspd"`
	Wspdx     pgtype.Float4      `json:"wspdx"`
	Srad      pgtype.Float4      `json:"srad"`
	Mslp      pgtype.Float4      `json:"mslp"`
	Hi        pgtype.Float4      `json:"hi"`
	Wchill    pgtype.Float4      `json:"wchill"`
	Timestamp pgtype.Timestamptz `json:"timestamp"`
}

type StationHealth struct {
	Vb1               pgtype.Float4      `json:"vb1"`
	Vb2               pgtype.Float4      `json:"vb2"`
	Curr              pgtype.Float4      `json:"curr"`
	Bp1               pgtype.Float4      `json:"bp1"`
	Bp2               pgtype.Float4      `json:"bp2"`
	Cm                pgtype.Text        `json:"cm"`
	Ss                pgtype.Int4        `json:"ss"`
	TempArq           pgtype.Float4      `json:"temp_arq"`
	RhArq             pgtype.Float4      `json:"rh_arq"`
	Fpm               pgtype.Text        `json:"fpm"`
	ErrorMsg          pgtype.Text        `json:"error_msg"`
	Message           pgtype.Text        `json:"message"`
	DataCount         pgtype.Int4        `json:"data_count"`
	DataStatus        pgtype.Text        `json:"data_status"`
	Timestamp         pgtype.Timestamptz `json:"timestamp"`
	MinutesDifference pgtype.Int4        `json:"minutes_difference"`
}

func (l Lufft) String(nVal int) string {
	var wspd, wspdx pgtype.Float4
	var rr pgtype.Int4
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

	// obsStr := ""
	var obsStrSlice []string
	for _, f := range []pgtype.Float4{l.Obs.Temp, l.Obs.Rh, l.Obs.Pres, wspd, wspdx} {
		if f.Valid {
			obsStrSlice = append(obsStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
		} else {
			obsStrSlice = append(obsStrSlice, "")
		}
	}
	if nVal == 20 || nVal == 24 {
		obsStrSlice = append(obsStrSlice, "0")
	}
	for _, f := range []pgtype.Float4{l.Obs.Wdir, l.Obs.Srad, l.Obs.Td, l.Obs.Wchill} {
		if f.Valid {
			obsStrSlice = append(obsStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
		} else {
			obsStrSlice = append(obsStrSlice, "")
		}
	}
	if rr.Valid {
		obsStrSlice = append(obsStrSlice, fmt.Sprintf("%d", rr.Int32))
	} else {
		obsStrSlice = append(obsStrSlice, "")
	}
	obsStr := strings.Join(obsStrSlice, "+")

	var hStrSlice []string
	if nVal == 23 || nVal == 24 {
		for _, f := range []pgtype.Float4{l.Health.Vb1, l.Health.Vb2, l.Health.Curr, l.Health.Bp1, l.Health.Bp2} {
			if f.Valid {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		if l.Health.Cm.Valid {
			hStrSlice = append(hStrSlice, l.Health.Cm.String)
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Ss.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", l.Health.Ss.Int32))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for _, f := range []pgtype.Float4{l.Health.TempArq, l.Health.RhArq} {
			if f.Valid {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		if l.Health.Fpm.Valid {
			hStrSlice = append(hStrSlice, l.Health.Fpm.String)
		} else {
			hStrSlice = append(hStrSlice, "")
		}
	} else if nVal == 19 {
		for _, f := range []pgtype.Float4{l.Health.TempArq, l.Health.RhArq} {
			if f.Valid {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		if l.Health.Ss.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", l.Health.Ss.Int32))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Vb1.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f#", math.Round(float64(l.Health.Vb1.Float32)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Bp1.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(l.Health.Bp1.Float32)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Fpm.Valid {
			hStrSlice = append(hStrSlice, l.Health.Fpm.String)
		} else {
			hStrSlice = append(hStrSlice, "")
		}
	} else if nVal == 20 {
		if l.Health.Ss.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", l.Health.Ss.Int32))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Vb1.Valid {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f#", math.Round(float64(l.Health.Vb1.Float32)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for _, f := range []pgtype.Float4{l.Health.Bp1, l.Health.TempArq, l.Health.RhArq} {
			if f.Valid {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(f.Float32)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		if l.Health.Fpm.Valid {
			hStrSlice = append(hStrSlice, l.Health.Fpm.String)
		} else {
			hStrSlice = append(hStrSlice, "")
		}
	}
	hStr := strings.Join(hStrSlice, "+")

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
			Cm: pgtype.Text{
				String: valStrs[16],
				Valid:  true,
			},
			Ss:      parseInt(valStrs[17]),
			TempArq: parseFloat(valStrs[18], false),
			RhArq:   parseFloat(valStrs[19], false),
			Fpm: pgtype.Text{
				String: valStrs[20],
				Valid:  true,
			},
		}
	} else if nVal == 19 {
		health = StationHealth{
			TempArq: parseFloat(valStrs[11], false),
			RhArq:   parseFloat(valStrs[12], false),
			Ss:      parseInt(valStrs[13]),
			Vb1:     parseFloat(strings.Split(valStrs[14], "#")[0], false),
			Bp1:     parseFloat(valStrs[15], false),
			Fpm: pgtype.Text{
				String: valStrs[16],
				Valid:  true,
			},
		}
	} else if nVal == 20 {
		health = StationHealth{
			Ss:      parseInt(valStrs[11]),
			Vb1:     parseFloat((valStrs[12])[:len(valStrs[12])-1], false),
			Bp1:     parseFloat(valStrs[13], false),
			TempArq: parseFloat(valStrs[14], false),
			RhArq:   parseFloat(valStrs[15], false),
			Fpm: pgtype.Text{String: valStrs[16],
				Valid: true},
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
	l.Health.Message = pgtype.Text{String: valStr,
		Valid: true,
	}

	l.Health.MinutesDifference = pgtype.Int4{
		Int32: int32(minutesDiff),
		Valid: true,
	}
	l.Health.ErrorMsg = pgtype.Text{String: errMsg,
		Valid: true,
	}
	l.Health.DataCount = pgtype.Int4{
		Int32: int32(dataCount),
		Valid: true,
	}
	l.Health.DataStatus = pgtype.Text{String: dataStatus,
		Valid: true,
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
			Temp: pgtype.Float4{
				Float32: RandomFloat(20, 35),
				Valid:   true,
			},
			Rh: pgtype.Float4{
				Float32: RandomFloat(0, 100),
				Valid:   true,
			},
			Pres: pgtype.Float4{
				Float32: RandomFloat(990, 1100),
				Valid:   true,
			},
			Wspd: pgtype.Float4{
				Float32: RandomFloat(20, 35),
				Valid:   true,
			},
			Wspdx: pgtype.Float4{
				Float32: RandomFloat(35, 50),
				Valid:   true,
			},
			Wdir: pgtype.Float4{
				Float32: RandomFloat(0, 359),
				Valid:   true,
			},
			Srad: pgtype.Float4{
				Float32: RandomFloat(0, 1000),
				Valid:   true,
			},
			Td: pgtype.Float4{
				Float32: RandomFloat(20, 35),
				Valid:   true,
			},
			Wchill: pgtype.Float4{
				Float32: RandomFloat(20, 35),
				Valid:   true,
			},
			Rr: pgtype.Float4{
				Float32: RandomFloat(0, 100),
				Valid:   true,
			},
			Timestamp: timestamp,
		},
		Health: StationHealth{
			Vb1: pgtype.Float4{
				Float32: RandomFloat(0, 20),
				Valid:   true,
			},
			Vb2: pgtype.Float4{
				Float32: RandomFloat(0, 20),
				Valid:   true,
			},
			Curr: pgtype.Float4{
				Float32: RandomFloat(0, 1),
				Valid:   true,
			},
			Bp1: pgtype.Float4{
				Float32: RandomFloat(0, 30),
				Valid:   true,
			},
			Bp2: pgtype.Float4{
				Float32: RandomFloat(0, 30),
				Valid:   true,
			},
			Cm: pgtype.Text{
				String: RandomString(6),
				Valid:  true,
			},
			Ss: pgtype.Int4{
				Int32: int32(RandomInt(0, 100)),
				Valid: true,
			},
			TempArq: pgtype.Float4{
				Float32: RandomFloat(20, 35),
				Valid:   true,
			},
			RhArq: pgtype.Float4{
				Float32: RandomFloat(0, 100),
				Valid:   true,
			},
			Fpm: pgtype.Text{
				String: RandomString(6),
				Valid:  true,
			},
			Timestamp: timestamp,
		},
	}
}

func parseFloat(s string, skipValidation bool) pgtype.Float4 {
	ret := pgtype.Float4{
		Float32: 0.0,
		Valid:   false,
	}

	val, err := strconv.ParseFloat(s, 32)
	if err != nil || (!skipValidation && val == missingValue) {
		return ret
	}

	ret.Float32 = float32(math.Round(val*100) / 100)
	ret.Valid = true

	return ret
}

func parseFloatWithCF(s string, skipValidation bool, cf float32) pgtype.Float4 {
	ret := parseFloat(s, skipValidation)

	if ret.Valid {
		ret.Float32 = float32(math.Round(float64(ret.Float32*cf)*100) / 100)
	}

	return ret
}

func parseInt(s string) pgtype.Int4 {
	ret := pgtype.Int4{
		Int32: 0,
		Valid: false,
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
