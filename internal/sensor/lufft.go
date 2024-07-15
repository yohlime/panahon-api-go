package sensor

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
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
	Pres      *float32  `json:"pres" fake:"{float32range:990,1100}"`
	Rr        *float32  `json:"rr" fake:"{float32range:0,100}"`
	Rh        *float32  `json:"rh" fake:"{float32range:0,100}"`
	Temp      *float32  `json:"temp" fake:"{float32range:20,37}"`
	Td        *float32  `json:"td" fake:"{float32range:15,40}"`
	Wdir      *float32  `json:"wdir" fake:"{float32range:0,359}"`
	Wspd      *float32  `json:"wspd" fake:"{float32range:20,35}"`
	Wspdx     *float32  `json:"wspdx" fake:"{float32range:35,50}"`
	Srad      *float32  `json:"srad" fake:"{float32range:0,1000}"`
	Mslp      *float32  `json:"mslp" fake:"-"`
	Hi        *float32  `json:"hi" fake:"-"`
	Wchill    *float32  `json:"wchill" fake:"{float32range:20,35}"`
	Timestamp time.Time `json:"timestamp"`
}

type StationHealth struct {
	Vb1               *float32  `json:"vb1" fake:"{float32range:0,20}"`
	Vb2               *float32  `json:"vb2" fake:"{float32range:0,20}"`
	Curr              *float32  `json:"curr" fake:"{float32range:0,1}"`
	Bp1               *float32  `json:"bp1" fake:"{float32range:0,30}"`
	Bp2               *float32  `json:"bp2" fake:"{float32range:0,30}"`
	Cm                string    `json:"cm" fake:"{lettern:6}"`
	Ss                *int32    `json:"ss" fake:"{number:0,100}"`
	TempArq           *float32  `json:"temp_arq" fake:"{float32range:20,35}"`
	RhArq             *float32  `json:"rh_arq" fake:"{float32range:0,100}"`
	Fpm               string    `json:"fpm" fake:"{lettern:6}"`
	ErrorMsg          string    `json:"error_msg"`
	Message           string    `json:"message" fake:"-"`
	DataCount         int32     `json:"data_count" fake:"-"`
	DataStatus        string    `json:"data_status" fake:"-"`
	Timestamp         time.Time `json:"timestamp"`
	MinutesDifference int32     `json:"minutes_difference" fake:"-"`
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

	// obsStr := ""
	var obsStrSlice []string
	for _, f := range []*float32{l.Obs.Temp, l.Obs.Rh, l.Obs.Pres, wspd, wspdx} {
		if f != nil {
			obsStrSlice = append(obsStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
		} else {
			obsStrSlice = append(obsStrSlice, "")
		}
	}
	if nVal == 20 || nVal == 24 {
		obsStrSlice = append(obsStrSlice, "0")
	}
	for _, f := range []*float32{l.Obs.Wdir, l.Obs.Srad, l.Obs.Td, l.Obs.Wchill} {
		if f != nil {
			obsStrSlice = append(obsStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
		} else {
			obsStrSlice = append(obsStrSlice, "")
		}
	}
	if rr != nil {
		obsStrSlice = append(obsStrSlice, fmt.Sprintf("%d", *rr))
	} else {
		obsStrSlice = append(obsStrSlice, "")
	}
	obsStr := strings.Join(obsStrSlice, "+")

	var hStrSlice []string
	if nVal == 23 || nVal == 24 {
		for _, f := range []*float32{l.Health.Vb1, l.Health.Vb2, l.Health.Curr, l.Health.Bp1, l.Health.Bp2} {
			if f != nil {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		hStrSlice = append(hStrSlice, l.Health.Cm)
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", *l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for _, f := range []*float32{l.Health.TempArq, l.Health.RhArq} {
			if f != nil {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	} else if nVal == 19 {
		for _, f := range []*float32{l.Health.TempArq, l.Health.RhArq} {
			if f != nil {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Vb1 != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f#", math.Round(float64(*l.Health.Vb1)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Bp1 != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*l.Health.Bp1)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	} else if nVal == 20 {
		if l.Health.Ss != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%d", l.Health.Ss))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		if l.Health.Vb1 != nil {
			hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f#", math.Round(float64(*l.Health.Vb1)*100)/100))
		} else {
			hStrSlice = append(hStrSlice, "")
		}
		for _, f := range []*float32{l.Health.Bp1, l.Health.TempArq, l.Health.RhArq} {
			if f != nil {
				hStrSlice = append(hStrSlice, fmt.Sprintf("%.2f", math.Round(float64(*f)*100)/100))
			} else {
				hStrSlice = append(hStrSlice, "")
			}
		}
		hStrSlice = append(hStrSlice, l.Health.Fpm)
	}
	hStr := strings.Join(hStrSlice, "+")

	timestampStr := l.Obs.Timestamp.Format(time.RFC3339)
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
			Vb1:     parseFloat(valStrs[11], false),
			Vb2:     parseFloat(valStrs[12], false),
			Curr:    parseFloat(valStrs[13], false),
			Bp1:     parseFloat(valStrs[14], false),
			Bp2:     parseFloat(valStrs[15], false),
			Cm:      valStrs[16],
			Ss:      parseInt(valStrs[17]),
			TempArq: parseFloat(valStrs[18], false),
			RhArq:   parseFloat(valStrs[19], false),
			Fpm:     valStrs[20],
		}
	} else if nVal == 19 {
		health = StationHealth{
			TempArq: parseFloat(valStrs[11], false),
			RhArq:   parseFloat(valStrs[12], false),
			Ss:      parseInt(valStrs[13]),
			Vb1:     parseFloat(strings.Split(valStrs[14], "#")[0], false),
			Bp1:     parseFloat(valStrs[15], false),
			Fpm:     valStrs[16],
		}
	} else if nVal == 20 {
		health = StationHealth{
			Ss:      parseInt(valStrs[11]),
			Vb1:     parseFloat((valStrs[12])[:len(valStrs[12])-1], false),
			Bp1:     parseFloat(valStrs[13], false),
			TempArq: parseFloat(valStrs[14], false),
			RhArq:   parseFloat(valStrs[15], false),
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

func parseFloat(s string, skipValidation bool) *float32 {
	val, err := strconv.ParseFloat(s, 32)
	if err != nil || (!skipValidation && val == missingValue) {
		return nil
	}

	f := float32(math.Round(val*100) / 100)
	return &f
}

func parseFloatWithCF(s string, skipValidation bool, cf float32) *float32 {
	ret := parseFloat(s, skipValidation)

	if ret != nil {
		v := float32(math.Round(float64(*ret*cf)*100) / 100)
		return &v
	}

	return nil
}

func parseInt(s string) *int32 {
	val, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return nil
	}

	v := int32(val)
	return &v
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
