// Package model contains Go types that mirror the Groundwork tRPC API
// response shapes. Only the fields the TUI actually uses are included.
//
// Note on superjson: The server uses superjson which serializes Date objects
// as {"$date": "2024-01-15T00:00:00.000Z"}. Fields typed as SuperJSONDate
// handle this transparently via custom UnmarshalJSON.
package model

import (
	"encoding/json"
	"strings"
	"time"
)

// SuperJSONDate is a date that may arrive as either a plain string ("2024-01-15")
// or a superjson Date object ({"$date":"2024-01-15T00:00:00.000Z"}).
type SuperJSONDate struct {
	Value string // always "YYYY-MM-DD" after unmarshalling
}

func (d *SuperJSONDate) UnmarshalJSON(b []byte) error {
	// Try plain string first
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		d.Value = strings.SplitN(s, "T", 2)[0]
		return nil
	}
	// Try superjson object: {"$date": "..."}
	var obj struct {
		Date string `json:"$date"`
	}
	if err := json.Unmarshal(b, &obj); err == nil && obj.Date != "" {
		d.Value = strings.SplitN(obj.Date, "T", 2)[0]
		return nil
	}
	d.Value = ""
	return nil
}

func (d SuperJSONDate) String() string { return d.Value }

// UnitType mirrors the Prisma UnitType enum.
type UnitType string

const (
	UnitTypeHours    UnitType = "HOURS"
	UnitTypeChapters UnitType = "CHAPTERS"
	UnitTypeSections UnitType = "SECTIONS"
	UnitTypeModules  UnitType = "MODULES"
	UnitTypeEpisodes UnitType = "EPISODES"
	UnitTypePages    UnitType = "PAGES"
	UnitTypeLessons  UnitType = "LESSONS"
	UnitTypeVideos   UnitType = "VIDEOS"
)

// Label returns a human-readable plural label for a unit type.
func (u UnitType) Label() string {
	return u.LabelFull()
}

// LabelFull returns the full plural label.
func (u UnitType) LabelFull() string {
	switch u {
	case UnitTypeHours:
		return "hours"
	case UnitTypeChapters:
		return "chapters"
	case UnitTypeSections:
		return "sections"
	case UnitTypeModules:
		return "modules"
	case UnitTypeEpisodes:
		return "episodes"
	case UnitTypePages:
		return "pages"
	case UnitTypeLessons:
		return "lessons"
	case UnitTypeVideos:
		return "videos"
	default:
		return "units"
	}
}

// MaterialStatus is a computed field — never stored in the DB.
type MaterialStatus string

const (
	StatusActive   MaterialStatus = "ACTIVE"
	StatusComplete MaterialStatus = "COMPLETE"
	StatusInactive MaterialStatus = "INACTIVE"
)

// --- dashboard.getOverview ---
// shape: { activeMaterials, completedMaterials, completionPct, streak, longestStreak }

type Overview struct {
	ActiveMaterials int     `json:"activeMaterials"`
	CompletedCount  int     `json:"completedMaterials"`
	CompletionPct   float64 `json:"completionPct"`
	CurrentStreak   int     `json:"streak"`
	LongestStreak   int     `json:"longestStreak"`
}

// --- dashboard.getChartData ---

type ChartDataPoint struct {
	Date  string  `json:"date"`
	Units float64 `json:"units"`
}

type ActiveMaterialSeries struct {
	MaterialID   string           `json:"materialId"`
	MaterialName string           `json:"materialName"`
	Data         []ChartDataPoint `json:"data"`
}

type ChartData struct {
	Daily           []ChartDataPoint       `json:"daily"`
	ActiveMaterials []ActiveMaterialSeries `json:"activeMaterials"`
}

// --- dashboard.getActiveMaterials ---
// shape: { id, name, url, totalUnits, unitType, weeklyUnitGoal,
//          skill: { id, name, color },
//          completedUnits, unitsThisWeek, pctComplete, projectedEndDate }

type ActiveMaterialSkill struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type ActiveMaterial struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	UnitType         UnitType            `json:"unitType"`
	TotalUnits       float64             `json:"totalUnits"`
	WeeklyUnitGoal   *int                `json:"weeklyUnitGoal"`
	URL              *string             `json:"url"`
	Skill            ActiveMaterialSkill `json:"skill"`
	CompletedUnits   float64             `json:"completedUnits"`
	UnitsThisWeek    float64             `json:"unitsThisWeek"`
	PctComplete      float64             `json:"pctComplete"`
	ProjectedEndDate *string             `json:"projectedEndDate"`
}

// SkillName convenience accessor.
func (a ActiveMaterial) SkillName() string { return a.Skill.Name }

// --- skill.getAll ---
// Actual shape: { id, name, categoryId, color, isArchived, priority,
//                 category: { id, name, color }, _count: { materials } }

type SkillCategory struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color"`
}

type SkillCount struct {
	Materials int `json:"materials"`
}

type Skill struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	CategoryID string        `json:"categoryId"`
	Color      *string       `json:"color"`
	IsArchived bool          `json:"isArchived"`
	Priority   int           `json:"priority"`
	Category   SkillCategory `json:"category"`
	Count      SkillCount    `json:"_count"`
}

// CategoryName is a convenience accessor.
func (s Skill) CategoryName() string { return s.Category.Name }

// MaterialCount is a convenience accessor.
func (s Skill) MaterialCount() int { return s.Count.Materials }

// --- category.getAll ---

type CategoryCount struct {
	Skills int `json:"skills"`
}

type Category struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Color      *string       `json:"color"`
	IsArchived bool          `json:"isArchived"`
	Count      CategoryCount `json:"_count"`
}

// SkillCount is a convenience accessor.
func (c Category) SkillCount() int { return c.Count.Skills }

// --- material.getAll ---
// Actual shape: { id, name, unitType, totalUnits, completedUnits, status,
//                 skillId, weeklyUnitGoal, url, startDate, completedDate,
//                 skill: { id, name, color, category: {...} },
//                 materialType: { id, name },
//                 _count: { progressLogs } }

type MaterialSkill struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Color    *string       `json:"color"`
	Category SkillCategory `json:"category"`
}

type MaterialType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MaterialCount struct {
	ProgressLogs int `json:"progressLogs"`
}

type Material struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	UnitType       UnitType       `json:"unitType"`
	TotalUnits     float64        `json:"totalUnits"`
	CompletedUnits float64        `json:"completedUnits"`
	Status         MaterialStatus `json:"status"`
	SkillID        string         `json:"skillId"`
	WeeklyUnitGoal *int           `json:"weeklyUnitGoal"`
	URL            *string        `json:"url"`
	StartDate      *SuperJSONDate `json:"startDate"`
	CompletedDate  *SuperJSONDate `json:"completedDate"`
	Skill          MaterialSkill  `json:"skill"`
	MaterialType   MaterialType   `json:"materialType"`
	Count          MaterialCount  `json:"_count"`
}

// SkillName convenience accessor.
func (m Material) SkillName() string { return m.Skill.Name }

// TypeName convenience accessor.
func (m Material) TypeName() string { return m.MaterialType.Name }

// --- progress.getAll ---
// Actual shape: { id, userId, materialId, date, units, notes, createdAt,
//                 material: { id, name, unitType } }

type ProgressMaterial struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	UnitType UnitType `json:"unitType"`
}

type ProgressLog struct {
	ID        string           `json:"id"`
	Date      SuperJSONDate    `json:"date"`
	Units     float64          `json:"units"`
	Notes     *string          `json:"notes"`
	CreatedAt time.Time        `json:"createdAt"`
	Material  ProgressMaterial `json:"material"`
}

// MaterialName convenience accessor.
func (p ProgressLog) MaterialName() string { return p.Material.Name }

// --- dashboard.getCategoryData ---

type CategorySkillSummary struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Color               *string `json:"color"`
	IsArchived          bool    `json:"isArchived"`
	MaterialCount       int     `json:"materialCount"`
	ActiveMaterialCount int     `json:"activeMaterialCount"`
	CompletedUnits      float64 `json:"completedUnits"`
	TotalUnits          float64 `json:"totalUnits"`
}

type CategoryActiveMaterial struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	CompletedUnits float64  `json:"completedUnits"`
	TotalUnits     float64  `json:"totalUnits"`
	UnitType       UnitType `json:"unitType"`
	SkillName      string   `json:"skillName"`
}

type CategoryDetail struct {
	Category struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Color      *string `json:"color"`
		IsArchived bool    `json:"isArchived"`
	} `json:"category"`
	TotalMaterials         int                      `json:"totalMaterials"`
	ActiveMaterialCount    int                      `json:"activeMaterialCount"`
	CompletedMaterialCount int                      `json:"completedMaterialCount"`
	PctCompleted           float64                  `json:"pctCompleted"`
	PctThisWeek            float64                  `json:"pctThisWeek"`
	SkillsSummary          []CategorySkillSummary   `json:"skillsSummary"`
	ActiveMaterials        []CategoryActiveMaterial `json:"activeMaterials"`
}

// --- dashboard.getSkillData ---

type SkillDetailMaterial struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Status           MaterialStatus `json:"status"`
	CompletedUnits   float64        `json:"completedUnits"`
	TotalUnits       float64        `json:"totalUnits"`
	UnitType         UnitType       `json:"unitType"`
	MaterialType     MaterialType   `json:"materialType"`
	WeeklyUnitGoal   *int           `json:"weeklyUnitGoal"`
	UnitsThisWeek    float64        `json:"unitsThisWeek"`
	PctComplete      float64        `json:"pctComplete"`
	ProjectedEndDate *string        `json:"projectedEndDate"`
}

type SkillDetail struct {
	Skill struct {
		ID         string        `json:"id"`
		Name       string        `json:"name"`
		Color      *string       `json:"color"`
		IsArchived bool          `json:"isArchived"`
		Category   SkillCategory `json:"category"`
	} `json:"skill"`
	TotalMaterials         int                   `json:"totalMaterials"`
	ActiveMaterialCount    int                   `json:"activeMaterialCount"`
	CompletedMaterialCount int                   `json:"completedMaterialCount"`
	PctCompleted           float64               `json:"pctCompleted"`
	PctThisWeek            float64               `json:"pctThisWeek"`
	ActiveMaterials        []SkillDetailMaterial `json:"activeMaterials"`
	AllMaterials           []SkillDetailMaterial `json:"allMaterials"`
}

// --- dashboard.getMaterialDetail ---

type MaterialDetailLog struct {
	ID    string        `json:"id"`
	Date  SuperJSONDate `json:"date"`
	Units float64       `json:"units"`
	Notes *string       `json:"notes"`
}

type MaterialDetailInfo struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	UnitType         UnitType       `json:"unitType"`
	TotalUnits       float64        `json:"totalUnits"`
	CompletedUnits   float64        `json:"completedUnits"`
	UnitsThisWeek    float64        `json:"unitsThisWeek"`
	PctComplete      float64        `json:"pctComplete"`
	Status           MaterialStatus `json:"status"`
	WeeklyUnitGoal   *int           `json:"weeklyUnitGoal"`
	URL              *string        `json:"url"`
	Notes            *string        `json:"notes"`
	StartDate        *SuperJSONDate `json:"startDate"`
	CompletedDate    *SuperJSONDate `json:"completedDate"`
	MaterialStreak   int            `json:"materialStreak"`
	ProjectedEndDate *string        `json:"projectedEndDate"`
	MaterialType     MaterialType   `json:"materialType"`
	Skill            MaterialSkill  `json:"skill"`
}

type MaterialDetail struct {
	Material     MaterialDetailInfo  `json:"material"`
	ProgressLogs []MaterialDetailLog `json:"progressLogs"`
}

// --- workout / fitness ---

// WorkoutType mirrors the Prisma WorkoutType enum.
type WorkoutType string

const (
	WorkoutTypeLifting WorkoutType = "LIFTING"
	WorkoutTypeRunning WorkoutType = "RUNNING"
)

// RunZone mirrors the Prisma RunZone enum.
type RunZone string

const (
	RunZoneZ1   RunZone = "Z1"
	RunZoneZ2   RunZone = "Z2"
	RunZoneZ3   RunZone = "Z3"
	RunZoneZ4   RunZone = "Z4"
	RunZoneZ5   RunZone = "Z5"
	RunZoneFree RunZone = "FREE"
)

// Exercise is a per-user lift exercise.
type Exercise struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	IsArchived bool   `json:"isArchived"`
	Priority   int    `json:"priority"`
}

// SessionLiftEntry is the raw lift entry returned by getSessions.
type SessionLiftEntry struct {
	Exercise struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"exercise"`
	WeightLbs float64 `json:"weightLbs"`
}

// SessionRunSegment is a single segment within a run returned by getSessions.
type SessionRunSegment struct {
	Zone            string  `json:"zone"`
	DistanceMiles   float64 `json:"distanceMiles"`
	DurationSeconds int     `json:"durationSeconds"`
}

// SessionRunEntry is the run-specific data returned inline by getSessions.
type SessionRunEntry struct {
	Segments []SessionRunSegment `json:"segments"`
}

// WorkoutSession is a single workout session (lift or run).
// LiftEntries and RunEntry are populated from the getSessions response and used
// to format the Details string and to pre-populate the edit form.
type WorkoutSession struct {
	ID              string             `json:"id"`
	Type            WorkoutType        `json:"type"`
	Date            SuperJSONDate      `json:"date"`
	DurationMinutes *int               `json:"durationMinutes"`
	Notes           *string            `json:"notes"`
	LiftEntries     []SessionLiftEntry `json:"liftEntries"`
	RunEntry        *SessionRunEntry   `json:"runEntry"`
	// Details is a pre-formatted summary string computed client-side.
	// Not part of the JSON response — populated by the API client after parsing.
	Details string `json:"-"`
}

// LiftRecord is one exercise+weight entry within a lifting session.
type LiftRecord struct {
	ExerciseID   string  `json:"exerciseId"`
	ExerciseName string  `json:"exerciseName"`
	WeightLbs    float64 `json:"weightLbs"`
}

// RunZonePace is the computed pace for a run zone.
type RunZonePace struct {
	Zone        RunZone `json:"zone"`
	PaceSeconds int     `json:"paceSeconds"`
}

// RunEntryDetail is the run-specific data embedded in a WorkoutSessionDetail.
type RunEntryDetail struct {
	DistanceMiles   float64       `json:"distanceMiles"`
	DurationSeconds int           `json:"durationSeconds"`
	ZonePaces       []RunZonePace `json:"zonePaces"`
}

// WorkoutSessionDetail embeds WorkoutSession with lift/run specifics.
type WorkoutSessionDetail struct {
	WorkoutSession
	LiftRecords []LiftRecord    `json:"liftRecords"`
	RunEntry    *RunEntryDetail `json:"runEntry"`
}

// WorkoutGoal is one goal entry (lifting or running sessions per week).
type WorkoutGoal struct {
	Type            WorkoutType `json:"type"`
	SessionsPerWeek int         `json:"sessionsPerWeek"`
}

// WorkoutStats holds the aggregated workout stats for the current week.
type WorkoutStats struct {
	LiftingThisWeek int `json:"thisWeekLiftSessions"`
	RunningThisWeek int `json:"thisWeekRunSessions"`
	LiftingGoal     int `json:"liftingGoal"`   // filled client-side from goals
	RunningGoal     int `json:"runningGoal"`   // filled client-side from goals
	LiftingStreak   int `json:"liftingStreak"` // not yet in API — reserved
	RunningStreak   int `json:"runningStreak"` // not yet in API — reserved
}

// LiftProgressEntry is one data point in a per-exercise weight-over-time series.
type LiftProgressEntry struct {
	Date      SuperJSONDate `json:"date"`
	WeightLbs float64       `json:"weightLbs"`
}

// LiftProgress is the weight-over-time series for one exercise.
type LiftProgress struct {
	ExerciseID   string              `json:"exerciseId"`
	ExerciseName string              `json:"exerciseName"`
	Entries      []LiftProgressEntry `json:"entries"`
}

// RunProgressEntry is one data point in the run distance/pace history.
type RunProgressEntry struct {
	Date            SuperJSONDate `json:"date"`
	DistanceMiles   float64       `json:"distanceMiles"`
	DurationSeconds int           `json:"durationSeconds"`
	PaceSeconds     int           `json:"paceSecondsPerMile"`
}

// RunProgress is the run distance/pace history — the server returns a plain array.
type RunProgress = []RunProgressEntry
