package sensor

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/emiliogozo/panahon-api-go/internal/util"
)

type Misol struct {
	StnID  int64
	Lat    float32
	Lon    float32
	Obs    StationObservation
	Health StationHealth
}

func NewMisolFromString(valStr string) (m *Misol, err error) {
	valStr = strings.TrimSpace(valStr)

	valStrs := strings.Split(valStr, ",")
	nVal := len(valStrs)

	m = new(Misol)
	if nVal < 27 {
		return nil, fmt.Errorf("invalid string: wrong length")
	}

	stnID := util.NewWrappedInt[int64](valStrs[0])
	if !stnID.Valid {
		return nil, fmt.Errorf("invalid string: wrong length")
	}
	m.StnID = stnID.Value
	m.Lon = util.NewWrappedFloat[float32](valStrs[1]).Round(2).Value
	m.Lat = util.NewWrappedFloat[float32](valStrs[2]).Round(2).Value

	timeInt := util.NewWrappedInt[int64](valStrs[3])
	if !timeInt.Valid {
		return nil, fmt.Errorf("invalid string: wrong length")
	}
	timeNow := time.Now()
	timestamp := time.Unix(timeInt.Value, 0)
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

	obsSlice := valStrs[4:15]
	m.Obs = StationObservation{
		Temp: util.NewWrappedFloat[float32](obsSlice[0]).Round(2).GetRef(),
		Rh:   util.NewWrappedFloat[float32](obsSlice[1]).Round(2).GetRef(),
		Pres: util.NewWrappedFloat[float32](obsSlice[2]).Round(2).Validate(func(v float32) bool {
			return v != missingValue
		}).GetRef(),
		Wspd:               util.NewWrappedFloat[float32](obsSlice[3]).Convert(1.0 / 3.6).Round(2).GetRef(),
		Wspdx:              util.NewWrappedFloat[float32](obsSlice[4]).Convert(1.0 / 3.6).Round(2).GetRef(),
		Wdir:               util.NewWrappedFloat[float32](obsSlice[5]).Round(2).GetRef(),
		Srad:               util.NewWrappedFloat[float32](obsSlice[6]).Round(2).GetRef(),
		Td:                 util.NewWrappedFloat[float32](obsSlice[7]).Round(2).GetRef(),
		Wchill:             util.NewWrappedFloat[float32](obsSlice[8]).Round(2).GetRef(),
		RainTips:           util.NewWrappedInt[int32](obsSlice[9]).GetRef(),
		RainCumulativeTips: util.NewWrappedInt[int32](obsSlice[10]).GetRef(),
		Timestamp:          timestamp,
	}

	hSlice := valStrs[15:27]
	m.Health = StationHealth{
		Vb1:  util.NewWrappedFloat[float32](hSlice[0]).Round(2).GetRef(),
		Vb2:  util.NewWrappedFloat[float32](hSlice[1]).Round(2).GetRef(),
		Curr: util.NewWrappedFloat[float32](hSlice[2]).Round(2).GetRef(),
		Bp1:  util.NewWrappedFloat[float32](hSlice[3]).Round(2).GetRef(),
		Bp2:  util.NewWrappedFloat[float32](hSlice[4]).Round(2).GetRef(),
		// Cm:      hSlice[5],
		// Ss:      parseInt(hSlice[6]),
		TempArq: util.NewWrappedFloat[float32](hSlice[7]).Round(2).GetRef(),
		RhArq:   util.NewWrappedFloat[float32](hSlice[8]).Round(2).GetRef(),
		// Fpm:     hSlice[9],
		Timestamp: timestamp,
	}

	var dataCount int32 = 0
	dataStatus := ""
	for _, v := range []*float32{
		m.Obs.Temp, m.Obs.Rh, m.Obs.Pres, m.Obs.Wspd, m.Obs.Wspdx,
		m.Obs.Wdir, m.Obs.Srad, m.Obs.Td, m.Obs.Wchill,
	} {
		b := 0
		if v != nil {
			dataCount++
			b = 1
		}

		dataStatus += fmt.Sprintf("%d", b)
	}
	for _, v := range []*int32{
		m.Obs.RainTips, m.Obs.RainCumulativeTips,
	} {
		b := 0
		if v != nil {
			dataCount++
			b = 1
		}

		dataStatus += fmt.Sprintf("%d", b)
	}

	m.Health.Message = valStr

	m.Health.MinutesDifference = int32(minutesDiff)
	m.Health.ErrorMsg = errMsg
	m.Health.DataCount = dataCount
	m.Health.DataStatus = dataStatus

	return m, nil
}
