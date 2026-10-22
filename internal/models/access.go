package models

import (
	"encoding/json"
	"slices"
	"time"

	db "github.com/emiliogozo/panahon-api-go/internal/db/sqlc"
)

const (
	AccessRequestPending  = "pending"
	AccessRequestApproved = "approved"
	AccessRequestDenied   = "denied"

	// AccessRequestCooldown is how long a user must wait after a denial
	// before submitting another request.
	AccessRequestCooldown = 24 * time.Hour
)

// UserAccess is a user's maximum allowed scope (the ceiling for their tokens).
// nil station_ids/variables means "all"; an empty list means "none".
type UserAccess struct {
	UserID         int64     `json:"user_id"`
	StationIDs     []int64   `json:"station_ids"`
	Variables      []string  `json:"variables"`
	MaxDaysHistory int       `json:"max_days_history"`
	UpdatedAt      time.Time `json:"updated_at"`
} //@name UserAccess

// DefaultUserAccess is the entitlement for a user without an explicit grant.
func DefaultUserAccess(userID int64) UserAccess {
	return UserAccess{
		UserID:         userID,
		StationIDs:     []int64{},
		Variables:      DefaultAPITokenVariables,
		MaxDaysHistory: DefaultMaxDaysHistory,
	}
}

// NewUserAccess builds the API model from the db row.
func NewUserAccess(row db.UserAccess) UserAccess {
	res := UserAccess{
		UserID:         row.UserID,
		MaxDaysHistory: int(row.MaxDaysHistory),
	}

	if row.StationIds != nil {
		_ = json.Unmarshal(row.StationIds, &res.StationIDs)
	}
	if row.Variables != nil {
		_ = json.Unmarshal(row.Variables, &res.Variables)
	}
	if row.UpdatedAt.Valid {
		res.UpdatedAt = row.UpdatedAt.Time
	}

	return res
}

// AccessRequest is a user's request to expand their entitlement.
type AccessRequest struct {
	ID             int64      `json:"id"`
	UserID         int64      `json:"user_id"`
	StationIDs     []int64    `json:"station_ids"`
	AllStations    bool       `json:"all_stations"`
	Variables      []string   `json:"variables"`
	AllVariables   bool       `json:"all_variables"`
	MaxDaysHistory *int       `json:"max_days_history"`
	Reason         string     `json:"reason"`
	Status         string     `json:"status"`
	ReviewedBy     *int64     `json:"reviewed_by"`
	ReviewNote     string     `json:"review_note"`
	ReviewedAt     *time.Time `json:"reviewed_at"`
	CreatedAt      time.Time  `json:"created_at"`
} //@name AccessRequest

// NewAccessRequest builds the API model from the db row.
func NewAccessRequest(row db.AccessRequest) AccessRequest {
	res := AccessRequest{
		ID:           row.ID,
		UserID:       row.UserID,
		AllStations:  row.AllStations,
		AllVariables: row.AllVariables,
		Status:       row.Status,
		Reason:       row.Reason.String,
	}

	if row.StationIds != nil {
		_ = json.Unmarshal(row.StationIds, &res.StationIDs)
	}
	if row.Variables != nil {
		_ = json.Unmarshal(row.Variables, &res.Variables)
	}
	if row.MaxDaysHistory.Valid {
		v := int(row.MaxDaysHistory.Int32)
		res.MaxDaysHistory = &v
	}
	if row.ReviewedBy.Valid {
		v := row.ReviewedBy.Int64
		res.ReviewedBy = &v
	}
	res.ReviewNote = row.ReviewNote.String
	if row.ReviewedAt.Valid {
		t := row.ReviewedAt.Time
		res.ReviewedAt = &t
	}
	if row.CreatedAt.Valid {
		res.CreatedAt = row.CreatedAt.Time
	}

	return res
}

// ToCreateReq converts a stored request back into mergeable input.
func (r AccessRequest) ToCreateReq() CreateAccessRequestReq {
	return CreateAccessRequestReq{
		StationIDs:     r.StationIDs,
		AllStations:    r.AllStations,
		Variables:      r.Variables,
		AllVariables:   r.AllVariables,
		MaxDaysHistory: r.MaxDaysHistory,
		Reason:         r.Reason,
	}
}

// CreateAccessRequestReq is the request body for a user's access request.
// Omitted fields mean "no change"; all_stations/all_variables request everything.
type CreateAccessRequestReq struct {
	StationIDs     []int64  `json:"station_ids"`
	AllStations    bool     `json:"all_stations"`
	Variables      []string `json:"variables"`
	AllVariables   bool     `json:"all_variables"`
	MaxDaysHistory *int     `json:"max_days_history"`
	Reason         string   `json:"reason"`
} //@name CreateAccessRequestParams

// ReviewAccessRequestReq is the body for approving/denying. The optional
// scope fields allow an admin to grant a subset (partial approval).
type ReviewAccessRequestReq struct {
	Note           string   `json:"note"`
	StationIDs     []int64  `json:"station_ids"`
	AllStations    bool     `json:"all_stations"`
	Variables      []string `json:"variables"`
	AllVariables   bool     `json:"all_variables"`
	MaxDaysHistory *int     `json:"max_days_history"`
} //@name ReviewAccessRequestParams

// UpdateUserAccessReq is the body for an admin directly setting an entitlement.
type UpdateUserAccessReq struct {
	StationIDs     []int64  `json:"station_ids"`
	AllStations    bool     `json:"all_stations"`
	Variables      []string `json:"variables"`
	AllVariables   bool     `json:"all_variables"`
	MaxDaysHistory int      `json:"max_days_history"`
} //@name UpdateUserAccessParams

// MergeRequest applies a request's additions to an entitlement (additive) and
// returns the resulting entitlement.
func MergeRequest(current UserAccess, req CreateAccessRequestReq) UserAccess {
	res := current

	switch {
	case req.AllStations:
		res.StationIDs = nil
	case len(req.StationIDs) > 0 && res.StationIDs != nil:
		res.StationIDs = unionInt64(res.StationIDs, req.StationIDs)
	}

	switch {
	case req.AllVariables:
		res.Variables = nil
	case len(req.Variables) > 0 && res.Variables != nil:
		res.Variables = unionStrings(res.Variables, req.Variables)
	}

	if req.MaxDaysHistory != nil {
		res.MaxDaysHistory = raiseHistory(res.MaxDaysHistory, *req.MaxDaysHistory)
	}

	return res
}

// ApplyReview applies an admin's (possibly partial) approval to an entitlement.
func ApplyReview(current UserAccess, req ReviewAccessRequestReq) UserAccess {
	res := current

	switch {
	case req.AllStations:
		res.StationIDs = nil
	case len(req.StationIDs) > 0 && res.StationIDs != nil:
		res.StationIDs = unionInt64(res.StationIDs, req.StationIDs)
	}

	switch {
	case req.AllVariables:
		res.Variables = nil
	case len(req.Variables) > 0 && res.Variables != nil:
		res.Variables = unionStrings(res.Variables, req.Variables)
	}

	if req.MaxDaysHistory != nil {
		res.MaxDaysHistory = raiseHistory(res.MaxDaysHistory, *req.MaxDaysHistory)
	}

	return res
}

// IntersectPermissions caps a token's permissions by a user's entitlement.
func IntersectPermissions(p APITokenPermissions, a UserAccess) APITokenPermissions {
	res := APITokenPermissions{
		StationIDs:     intersectInt64Nullable(p.StationIDs, a.StationIDs),
		Variables:      intersectStringsNullable(resolveAll(p.Variables), resolveAll(a.Variables)),
		MaxDaysHistory: minHistory(p.MaxDaysHistory, a.MaxDaysHistory),
	}

	// Preserve "all" (nil) semantics produced by a nil-on-either-side intersection.
	if p.StationIDs == nil && a.StationIDs == nil {
		res.StationIDs = nil
	}
	return res
}

func unionInt64(a, b []int64) []int64 {
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func unionStrings(a, b []string) []string {
	out := slices.Clone(a)
	for _, v := range b {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func intersectInt64Nullable(a, b []int64) []int64 {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := make([]int64, 0, len(a))
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}

func intersectStringsNullable(a, b []string) []string {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	out := make([]string, 0, len(a))
	for _, v := range a {
		if slices.Contains(b, v) {
			out = append(out, v)
		}
	}
	return out
}

func resolveAll(vars []string) []string {
	for _, v := range vars {
		if v == APITokenAllVariables {
			return nil
		}
	}
	return vars
}

// capHistory maps 0 (unset) to the default; negatives (unlimited) pass through.
func capHistory(days int) int {
	if days == 0 {
		return DefaultMaxDaysHistory
	}
	return days
}

// raiseHistory merges a requested history value (raise-only; -1 = unlimited).
func raiseHistory(current, requested int) int {
	if current < 0 || requested < 0 {
		return -1
	}
	if requested > current {
		return requested
	}
	return current
}

// minHistory is the tighter of two history caps, where a negative means unlimited.
func minHistory(a, b int) int {
	a, b = capHistory(a), capHistory(b)
	switch {
	case a < 0:
		return b
	case b < 0:
		return a
	case a < b:
		return a
	default:
		return b
	}
}

// MarshalUserAccess encodes the nullable JSONB columns of an entitlement.
func MarshalUserAccess(a UserAccess) (stationIDs, variables []byte) {
	return marshalNullable(a.StationIDs), marshalNullable(a.Variables)
}

// Permissions expresses the entitlement as token-style permissions.
func (a UserAccess) Permissions() APITokenPermissions {
	return APITokenPermissions{
		StationIDs:     a.StationIDs,
		Variables:      a.Variables,
		MaxDaysHistory: a.MaxDaysHistory,
	}
}

func marshalNullable[T any](v []T) []byte {
	if v == nil {
		return nil
	}
	b, _ := json.Marshal(v)
	return b
}
