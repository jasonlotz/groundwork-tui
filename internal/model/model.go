// Package model contains Go types that mirror the Groundwork tRPC API
// response shapes. Only the fields the TUI actually uses are included.
package model

import "time"

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

// UnitLabel returns a human-readable plural label for a unit type.
func (u UnitType) Label() string {
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

type Overview struct {
	ActiveMaterials int     `json:"activeMaterials"`
	CompletionPct   float64 `json:"completionPct"`
	CompletedCount  int     `json:"completedCount"`
	CurrentStreak   int     `json:"currentStreak"`
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

type ActiveMaterial struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	UnitType         UnitType `json:"unitType"`
	TotalUnits       float64  `json:"totalUnits"`
	CompletedUnits   float64  `json:"completedUnits"`
	WeeklyUnitGoal   *int     `json:"weeklyUnitGoal"`
	UnitsThisWeek    float64  `json:"unitsThisWeek"`
	PctComplete      float64  `json:"pctComplete"`
	ProjectedEndDate *string  `json:"projectedEndDate"`
	SkillName        string   `json:"skillName"`
	URL              *string  `json:"url"`
}

// --- skill.getAll ---

type Skill struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	CategoryID   string  `json:"categoryId"`
	CategoryName string  `json:"categoryName"`
	Color        *string `json:"color"`
	IsArchived   bool    `json:"isArchived"`
	Priority     int     `json:"priority"`
}

// --- category.getAll ---

type Category struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Color      *string `json:"color"`
	IsArchived bool    `json:"isArchived"`
	SkillCount int     `json:"skillCount"`
}

// --- material.getAll ---

type Material struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	UnitType       UnitType       `json:"unitType"`
	TotalUnits     float64        `json:"totalUnits"`
	CompletedUnits float64        `json:"completedUnits"`
	Status         MaterialStatus `json:"status"`
	SkillID        string         `json:"skillId"`
	SkillName      string         `json:"skillName"`
	TypeName       string         `json:"typeName"`
	URL            *string        `json:"url"`
	StartDate      *string        `json:"startDate"`
	CompletedDate  *string        `json:"completedDate"`
}

// --- progress.getAll ---

type ProgressLog struct {
	ID           string    `json:"id"`
	MaterialID   string    `json:"materialId"`
	MaterialName string    `json:"materialName"`
	Date         string    `json:"date"`
	Units        float64   `json:"units"`
	Notes        *string   `json:"notes"`
	CreatedAt    time.Time `json:"createdAt"`
}
