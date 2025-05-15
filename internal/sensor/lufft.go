package sensor

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/emiliogozo/panahon-api-go/internal/util"
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
	Pres               *float32  `json:"pres"`
	Rr                 *float32  `json:"rr"`
	RainTips           *int32    `json:"rain_tips"`
	RainCumulativeTips *int32    `json:"rain_cumulative_tips"`
	Rh                 *float32  `json:"rh"`
	Temp               *float32  `json:"temp"`
	Td                 *float32  `json:"td"`
	Wdir               *float32  `json:"wdir"`
	Wspd               *float32  `json:"wspd"`
	Wspdx              *float32  `json:"wspdx"`
	Srad               *float32  `json:"srad"`
	Mslp               *float32  `json:"mslp"`
	Hi                 *float32  `json:"hi"`
	Wchill             *float32  `json:"wchill"`
	Timestamp          time.Time `json:"timestamp"`
}

type StationHealth struct {
	Vb1               *float32  `json:"vb1"`
	Vb2               *float32  `json:"vb2"`
	Curr              *float32  `json:"curr"`
	Bp1               *float32  `json:"bp1"`
	Bp2               *float32  `json:"bp2"`
	Cm                string    `json:"cm"`
	Ss                *int32    `json:"ss"`
	TempArq           *float32  `json:"temp_arq"`
	RhArq             *float32  `json:"rh_arq"`
	Fpm               string    `json:"fpm"`
	ErrorMsg          string    `json:"error_msg"`
	Message           string    `json:"message"`
	DataCount         int32     `json:"data_count"`
	DataStatus        string    `json:"data_status"`
	Timestamp         time.Time `json:"timestamp"`
	MinutesDifference int32     `json:"minutes_difference"`
}

func (l Lufft) String(nVal int) string {
	var wspd, wspdx *float32
	var rr *int32
	if l.Obs.Wspd != nil {
		v := *l.Obs.Wspd * 3.6
		wspd = &v
	}
	if l.Obs.Wspdx != nil {
		v := *l.Obs.Wspdx * 3.6
		wspdx = &v
	}
	if l.Obs.Rr != nil {
		v := int32(*l.Obs.Rr / (0.2 * 6.0))
		rr = &v
	}

	var obsStrSlice []string
	for _, f := range []*float32{l.Obs.Temp, l.Obs.Rh, l.Obs.Pres, wspd, wspdx} {
		pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
		obsStrSlice = append(obsStrSlice, pf.String())
	}
	if nVal == 20 || nVal == 24 {
		obsStrSlice = append(obsStrSlice, "0")
	}
	for _, f := range []*float32{l.Obs.Wdir, l.Obs.Srad, l.Obs.Td, l.Obs.Wchill} {
		pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
		obsStrSlice = append(obsStrSlice, pf.String())
	}
	if rr != nil {
		obsStrSlice = append(obsStrSlice, fmt.Sprintf("%d", *rr))
	} else {
		obsStrSlice = append(obsStrSlice, "")
	}
	obsStr := strings.Join(obsStrSlice, "+")

	var hStrSlice []string
	switch nVal {
	case 23, 24:
		for _, f := range []*float32{l.Health.Vb1, l.Health.Vb2, l.Health.Curr, l.Health.Bp1, l.Health.Bp2} {
			pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
			hStrSlice = append(hStrSlice, pf.String())
		}
		hStrSlice = append(hStrSlice, l.Health.Cm)
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", *l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for _, f := range []*float32{l.Health.TempArq, l.Health.RhArq} {
			pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
			hStrSlice = append(hStrSlice, pf.String())
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	case 19:
		for _, f := range []*float32{l.Health.TempArq, l.Health.RhArq} {
			pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
			hStrSlice = append(hStrSlice, pf.String())
		}
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", *l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for i, f := range []*float32{l.Health.Vb1, l.Health.Bp1} {
			pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
			vStr := pf.String()
			if i == 0 {
				vStr += "#"
			}
			hStrSlice = append(hStrSlice, vStr)
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	case 20:
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", *l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for i, f := range []*float32{l.Health.Vb1, l.Health.Bp1, l.Health.TempArq, l.Health.RhArq} {
			pf := util.WrappedFloat[float32]{Value: *f, Valid: f != nil, Decimals: 2}
			vStr := pf.String()
			if i == 0 {
				vStr += "#"
			}
			hStrSlice = append(hStrSlice, vStr)
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	}
	hStr := strings.Join(hStrSlice, "+")

	timestampStr := l.Obs.Timestamp.Format(time.RFC3339)
	switch nVal {
	case 23, 24:
		tStr := fmt.Sprintf("%s%s%s/%s%s%s",
			timestampStr[0:4], timestampStr[5:7], timestampStr[8:10],
			timestampStr[11:13], timestampStr[14:16], timestampStr[17:19],
		)
		return fmt.Sprintf("0+%s+0+%s+%s", obsStr, hStr, tStr)
	case 19, 20:
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
	timestamp := parseTimestampTz(valStrs[nVal-2], "Asia/Manila")
	errMsg := ""
	minutesDiff := 0.0

	if !timestamp.IsZero() {
		minutesDiff = timeNow.Sub(timestamp).Minutes()
		if minutesDiff < minMinutesThresh {
			errMsg = fmt.Sprintf("timestamp is %f minutes behind", math.Abs(minutesDiff))
			timestamp = timeNow
		} else if minutesDiff > maxMinutesThresh {
			errMsg = fmt.Sprintf("timestamp is %f minutes ahead", minutesDiff)
			timestamp = timeNow
		}
	}

	if nVal == 20 || nVal == 24 {
		_v := make([]string, 0)
		_v = append(_v, valStrs[:5]...)
		valStrs = append(_v, valStrs[6:]...)
	}

	if nVal >= 19 {
		l.Obs = StationObservation{
			Temp: util.NewWrappedFloat[float32](valStrs[0]).Round(2).GetRef(),
			Rh:   util.NewWrappedFloat[float32](valStrs[1]).Round(2).GetRef(),
			Pres: util.NewWrappedFloat[float32](valStrs[2]).Round(2).Validate(func(v float32) bool {
				return v != missingValue
			}).GetRef(),
			Wspd:      util.NewWrappedFloat[float32](valStrs[3]).Convert(1.0 / 3.6).Round(2).GetRef(),
			Wspdx:     util.NewWrappedFloat[float32](valStrs[4]).Convert(1.0 / 3.6).Round(2).GetRef(),
			Wdir:      util.NewWrappedFloat[float32](valStrs[5]).Round(2).GetRef(),
			Srad:      util.NewWrappedFloat[float32](valStrs[6]).Round(2).GetRef(),
			Td:        util.NewWrappedFloat[float32](valStrs[7]).Round(2).GetRef(),
			Wchill:    util.NewWrappedFloat[float32](valStrs[8]).Round(2).GetRef(),
			Rr:        util.NewWrappedFloat[float32](valStrs[9]).Convert(0.2 * 6.0).Round(2).GetRef(),
			Timestamp: timestamp,
		}
	}

	var health StationHealth
	switch nVal {
	case 23, 24:
		health = StationHealth{
			Vb1:     util.NewWrappedFloat[float32](valStrs[11]).Round(2).GetRef(),
			Vb2:     util.NewWrappedFloat[float32](valStrs[12]).Round(2).GetRef(),
			Curr:    util.NewWrappedFloat[float32](valStrs[13]).Round(2).GetRef(),
			Bp1:     util.NewWrappedFloat[float32](valStrs[14]).Round(2).GetRef(),
			Bp2:     util.NewWrappedFloat[float32](valStrs[15]).Round(2).GetRef(),
			Cm:      valStrs[16],
			Ss:      util.NewWrappedInt[int32](valStrs[17]).GetRef(),
			TempArq: util.NewWrappedFloat[float32](valStrs[18]).Round(2).GetRef(),
			RhArq:   util.NewWrappedFloat[float32](valStrs[19]).Round(2).GetRef(),
			Fpm:     valStrs[20],
		}
	case 19:
		health = StationHealth{
			TempArq: util.NewWrappedFloat[float32](valStrs[11]).Round(2).GetRef(),
			RhArq:   util.NewWrappedFloat[float32](valStrs[12]).Round(2).GetRef(),
			Ss:      util.NewWrappedInt[int32](valStrs[13]).GetRef(),
			Vb1:     util.NewWrappedFloat[float32](strings.Split(valStrs[14], "#")[0]).Round(2).GetRef(),
			Bp1:     util.NewWrappedFloat[float32](valStrs[15]).Round(2).GetRef(),
			Fpm:     valStrs[16],
		}
	case 20:
		health = StationHealth{
			Ss:      util.NewWrappedInt[int32](valStrs[11]).GetRef(),
			Vb1:     util.NewWrappedFloat[float32]((valStrs[12])[:len(valStrs[12])-1]).Round(2).GetRef(),
			Bp1:     util.NewWrappedFloat[float32](valStrs[13]).Round(2).GetRef(),
			TempArq: util.NewWrappedFloat[float32](valStrs[14]).Round(2).GetRef(),
			RhArq:   util.NewWrappedFloat[float32](valStrs[15]).Round(2).GetRef(),
			Fpm:     valStrs[16],
		}
	}

	dataCount := 0
	dataStatus := ""
	for _, v := range []*float32{
		l.Obs.Temp, l.Obs.Rh, l.Obs.Pres, l.Obs.Wspd, l.Obs.Wspdx,
		l.Obs.Wdir, l.Obs.Srad, l.Obs.Td, l.Obs.Wchill, l.Obs.Rr,
	} {
		b := 0
		if v != nil {
			dataCount++
			b = 1
		}

		dataStatus += fmt.Sprintf("%d", b)
	}

	l.Health = health
	l.Health.Timestamp = timestamp
	l.Health.Message = valStr

	l.Health.MinutesDifference = int32(minutesDiff)
	l.Health.ErrorMsg = errMsg
	l.Health.DataCount = int32(dataCount)
	l.Health.DataStatus = dataStatus

	return l, nil
}

func RandomLufft(timestamp time.Time) Lufft {
	obs := StationObservation{
		Pres:      util.ToRef(util.RandomFloat[float32](990, 1100)),
		Rr:        util.ToRef(util.RandomFloat[float32](0, 100)),
		Rh:        util.ToRef(util.RandomFloat[float32](0, 100)),
		Temp:      util.ToRef(util.RandomFloat[float32](20, 37)),
		Td:        util.ToRef(util.RandomFloat[float32](15, 40)),
		Wdir:      util.ToRef(util.RandomFloat[float32](0, 359)),
		Wspd:      util.ToRef(util.RandomFloat[float32](20, 35)),
		Wspdx:     util.ToRef(util.RandomFloat[float32](35, 50)),
		Srad:      util.ToRef(util.RandomFloat[float32](0, 1000)),
		Wchill:    util.ToRef(util.RandomFloat[float32](20, 35)),
		Timestamp: timestamp,
	}
	health := StationHealth{
		Vb1:       util.ToRef(util.RandomFloat[float32](0, 20)),
		Vb2:       util.ToRef(util.RandomFloat[float32](0, 20)),
		Curr:      util.ToRef(util.RandomFloat[float32](0, 1)),
		Bp1:       util.ToRef(util.RandomFloat[float32](0, 30)),
		Bp2:       util.ToRef(util.RandomFloat[float32](0, 30)),
		Cm:        gofakeit.LetterN(6),
		Ss:        util.ToRef(util.RandomInt[int32](0, 100)),
		TempArq:   util.ToRef(util.RandomFloat[float32](20, 35)),
		RhArq:     util.ToRef(util.RandomFloat[float32](0, 100)),
		Fpm:       gofakeit.LetterN(6),
		Timestamp: timestamp,
	}
	return Lufft{Obs: obs, Health: health}
}

func parseTimestampTz(dateStr string, tz string) time.Time {
	formats := []string{
		"06:01:02:15:04:05",   // YY:MM:DD:HH:MM:SS
		"2006:01:02:15:04:05", // YYYY:MM:DD:HH:MM:SS
		"060102/150405",       // YYMMDD/HHMMSS
		"20060102/150405",     // YYYYMMDD/HHMMSS
	}

	if tz == "" {
		tz = "Asia/Manila"
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}
	}

	var t time.Time
	for _, format := range formats {
		t, err = time.ParseInLocation(format, dateStr, loc)
		if err == nil {
			return t
		}
	}

	return time.Time{}
}
