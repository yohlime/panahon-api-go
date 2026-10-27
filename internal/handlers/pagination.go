package handlers

const (
	defaultStationPageSize     int32 = 100
	maxStationPageSize         int32 = 1000
	defaultObservationPageSize int32 = 100
	maxObservationPageSize     int32 = 5000
)

func normalizePage(page int32) int32 {
	if page < 1 {
		return 1
	}
	return page
}

func normalizePageSize(requested, fallback, maximum int32) int32 {
	if requested <= 0 {
		requested = fallback
	}
	if requested > maximum {
		requested = maximum
	}
	return requested
}

func observationPageSize(perPage, limit int32) int32 {
	if limit > 0 {
		return normalizePageSize(limit, defaultObservationPageSize, maxObservationPageSize)
	}
	return normalizePageSize(perPage, defaultObservationPageSize, maxObservationPageSize)
}
