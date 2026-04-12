// Package api provides a typed HTTP client for the Groundwork tRPC API.
//
// tRPC 11 protocol:
//   - Queries:   GET /api/trpc/<procedure>?input=<url-encoded-json>
//     Response: {"result":{"data":{"json":<output>}}}
//   - Mutations: POST /api/trpc/<procedure>  body: {"json":<input>}
//     Response: {"result":{"data":{"json":<output>}}}
//
// Auth: Authorization: Bearer <api_key>
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jasonlotz/groundwork-tui/internal/model"
)

// debugLogger is initialized once on first use and writes to
// ~/.cache/groundwork-tui/debug.log. A nil value means initialization
// failed silently (e.g. no home dir), in which case debug logging is a no-op.
var (
	debugLogger     *log.Logger
	debugLoggerOnce sync.Once
)

// initDebugLogger opens (or creates) the debug log file and sets debugLogger.
// Called once via sync.Once. Errors are silently swallowed so they never
// surface to the user.
func initDebugLogger() {
	dir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	logDir := filepath.Join(dir, "groundwork-tui")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(logDir, "debug.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	// Note: f is intentionally never closed — it lives for the duration of the process.
	debugLogger = log.New(f, "", log.LstdFlags)
}

// debugLog writes a message to ~/.cache/groundwork-tui/debug.log.
// The log file is opened once on first use. Errors are silently ignored.
func debugLog(format string, args ...any) {
	debugLoggerOnce.Do(initDebugLogger)
	if debugLogger != nil {
		debugLogger.Printf(format, args...)
	}
}

// Client is a typed tRPC HTTP client.
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

// --- low-level tRPC plumbing ---

type trpcRequest struct {
	JSON any `json:"json"`
}

// trpcRequestWithMeta is used when the input contains Date fields that must be
// described in SuperJSON's meta.values map so tRPC can reconstruct them as JS
// Date objects on the server side.
//
// SuperJSON wire format for a Date:
//
//	{"json": {"startDate": "2025-01-01T00:00:00.000Z"},
//	 "meta": {"values": {"startDate": ["Date"]}}}
type trpcRequestWithMeta struct {
	JSON any                `json:"json"`
	Meta *trpcSuperJSONMeta `json:"meta,omitempty"`
}

type trpcSuperJSONMeta struct {
	Values map[string][]string `json:"values"`
}

type trpcResponse[T any] struct {
	Result struct {
		Data struct {
			JSON T `json:"json"`
		} `json:"data"`
	} `json:"result"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) doRequest(req *http.Request) ([]byte, error) {
	req.Header.Set("x-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("unauthorized — check your API key")
	}
	if resp.StatusCode >= 400 {
		debugLog("HTTP %d %s: %s", resp.StatusCode, req.URL.Path, string(raw))
		return nil, fmt.Errorf("server error %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func parseResponse[T any](raw []byte) (T, error) {
	var zero T
	var result trpcResponse[T]
	if err := json.Unmarshal(raw, &result); err != nil {
		return zero, fmt.Errorf("unmarshal response: %w", err)
	}
	if result.Error != nil {
		return zero, fmt.Errorf("tRPC error: %s", result.Error.Message)
	}
	return result.Result.Data.JSON, nil
}

// query sends a GET request for a tRPC query procedure.
// Input is serialized as ?input=<url-encoded-json-wrapped-in-{"json":...}>
func query[T any](c *Client, procedure string, input any) (T, error) {
	var zero T

	inputJSON, err := json.Marshal(trpcRequest{JSON: input})
	if err != nil {
		return zero, fmt.Errorf("marshal input: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s?input=%s",
		c.baseURL, procedure, url.QueryEscape(string(inputJSON)))

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// mutation sends a POST request for a tRPC mutation procedure.
func mutation[T any](c *Client, procedure string, input any) (T, error) {
	var zero T

	body, err := json.Marshal(trpcRequest{JSON: input})
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s", c.baseURL, procedure)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// mutationWithMeta is like mutation but accepts a trpcRequestWithMeta so that
// SuperJSON meta (e.g. date type hints) is included in the request body.
func mutationWithMeta[T any](c *Client, procedure string, body trpcRequestWithMeta) (T, error) {
	var zero T

	b, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/trpc/%s", c.baseURL, procedure)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	raw, err := c.doRequest(req)
	if err != nil {
		return zero, err
	}
	return parseResponse[T](raw)
}

// superJSONDates builds a trpcRequestWithMeta wrapping input, adding a
// meta.values entry for each non-empty date field path so tRPC/SuperJSON
// reconstructs them as JS Date objects on the server.
// datePaths are the dot-notation key paths of date fields (e.g. "startDate").
func superJSONDates(input any, datePaths ...string) trpcRequestWithMeta {
	meta := &trpcSuperJSONMeta{Values: make(map[string][]string)}
	for _, p := range datePaths {
		meta.Values[p] = []string{"Date"}
	}
	return trpcRequestWithMeta{JSON: input, Meta: meta}
}

// --- typed procedures ---

// GetOverview calls dashboard.getOverview.
func (c *Client) GetOverview() (*model.Overview, error) {
	out, err := query[model.Overview](c, "dashboard.getOverview", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetChartData calls dashboard.getChartData.
func (c *Client) GetChartData() (*model.ChartData, error) {
	out, err := query[model.ChartData](c, "dashboard.getChartData", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetActiveMaterials calls dashboard.getActiveMaterials.
func (c *Client) GetActiveMaterials() ([]model.ActiveMaterial, error) {
	out, err := query[[]model.ActiveMaterial](c, "dashboard.getActiveMaterials", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getAllSkillsInput struct {
	IncludeArchived bool `json:"includeArchived,omitempty"`
}

// GetAllSkills calls skill.getAll. Pass includeArchived=true to include archived skills.
func (c *Client) GetAllSkills(includeArchived bool) ([]model.Skill, error) {
	out, err := query[[]model.Skill](c, "skill.getAll", getAllSkillsInput{IncludeArchived: includeArchived})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getAllCategoriesInput struct {
	IncludeArchived bool `json:"includeArchived,omitempty"`
}

// GetAllCategories calls category.getAll. Pass includeArchived=true to include archived categories.
func (c *Client) GetAllCategories(includeArchived bool) ([]model.Category, error) {
	out, err := query[[]model.Category](c, "category.getAll", getAllCategoriesInput{IncludeArchived: includeArchived})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getAllMaterialsInput struct {
	Status *string `json:"status,omitempty"`
}

// GetAllMaterials calls material.getAll with an optional status=ACTIVE filter.
func (c *Client) GetAllMaterials(activeOnly bool) ([]model.Material, error) {
	var input getAllMaterialsInput
	if activeOnly {
		s := "ACTIVE"
		input.Status = &s
	}
	out, err := query[[]model.Material](c, "material.getAll", input)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type getProgressInput struct {
	MaterialID *string `json:"materialId,omitempty"`
}

// GetAllProgress calls learningLog.getAll.
func (c *Client) GetAllProgress(materialID *string) ([]model.ProgressLog, error) {
	out, err := query[[]model.ProgressLog](c, "learningLog.getAll", getProgressInput{MaterialID: materialID})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type logUnitsInput struct {
	MaterialID string  `json:"materialId"`
	Date       string  `json:"date"`
	Units      float64 `json:"units"`
	Notes      *string `json:"notes,omitempty"`
}

type deleteProgressEntryInput struct {
	ID string `json:"id"`
}

// DeleteProgressEntry calls learningLog.deleteEntry (mutation).
func (c *Client) DeleteProgressEntry(id string) error {
	_, err := mutation[struct{}](c, "learningLog.deleteEntry", deleteProgressEntryInput{ID: id})
	return err
}

// LogUnits calls progress.logUnits (mutation).
func (c *Client) LogUnits(materialID, date string, units float64, notes *string) error {
	input := logUnitsInput{
		MaterialID: materialID,
		Date:       date + "T00:00:00.000Z",
		Units:      units,
		Notes:      notes,
	}
	_, err := mutationWithMeta[struct{}](c, "learningLog.logUnits", superJSONDates(input, "date"))
	return err
}

type getCategoryDataInput struct {
	CategoryID string `json:"categoryId"`
}

// GetCategoryData calls dashboard.getCategoryData.
func (c *Client) GetCategoryData(categoryID string) (*model.CategoryDetail, error) {
	out, err := query[model.CategoryDetail](c, "dashboard.getCategoryData", getCategoryDataInput{CategoryID: categoryID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getSkillDataInput struct {
	SkillID string `json:"skillId"`
}

// GetSkillData calls dashboard.getSkillData.
func (c *Client) GetSkillData(skillID string) (*model.SkillDetail, error) {
	out, err := query[model.SkillDetail](c, "dashboard.getSkillData", getSkillDataInput{SkillID: skillID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getMaterialDetailInput struct {
	MaterialID string `json:"materialId"`
}

// GetMaterialDetail calls dashboard.getMaterialDetail.
func (c *Client) GetMaterialDetail(materialID string) (*model.MaterialDetail, error) {
	out, err := query[model.MaterialDetail](c, "dashboard.getMaterialDetail", getMaterialDetailInput{MaterialID: materialID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// --- category mutations ---

type categoryCreateInput struct {
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// CreateCategory calls category.create.
func (c *Client) CreateCategory(name string, color *string) error {
	_, err := mutation[struct{}](c, "category.create", categoryCreateInput{Name: name, Color: color})
	return err
}

type categoryUpdateInput struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

// UpdateCategory calls category.update.
func (c *Client) UpdateCategory(id, name string, color *string) error {
	_, err := mutation[struct{}](c, "category.update", categoryUpdateInput{ID: id, Name: name, Color: color})
	return err
}

type categoryIDInput struct {
	ID string `json:"id"`
}

// ArchiveCategory calls category.archive.
func (c *Client) ArchiveCategory(id string) error {
	_, err := mutation[struct{}](c, "category.archive", categoryIDInput{ID: id})
	return err
}

// UnarchiveCategory calls category.unarchive.
func (c *Client) UnarchiveCategory(id string) error {
	_, err := mutation[struct{}](c, "category.unarchive", categoryIDInput{ID: id})
	return err
}

// DeleteCategory calls category.delete.
func (c *Client) DeleteCategory(id string) error {
	_, err := mutation[struct{}](c, "category.delete", categoryIDInput{ID: id})
	return err
}

// --- skill mutations ---

type skillCreateInput struct {
	Name       string  `json:"name"`
	CategoryID string  `json:"categoryId"`
	Color      *string `json:"color,omitempty"`
}

// CreateSkill calls skill.create.
func (c *Client) CreateSkill(name, categoryID string, color *string) error {
	_, err := mutation[struct{}](c, "skill.create", skillCreateInput{Name: name, CategoryID: categoryID, Color: color})
	return err
}

type skillUpdateInput struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CategoryID string  `json:"categoryId"`
	Color      *string `json:"color,omitempty"`
}

// UpdateSkill calls skill.update.
func (c *Client) UpdateSkill(id, name, categoryID string, color *string) error {
	_, err := mutation[struct{}](c, "skill.update", skillUpdateInput{ID: id, Name: name, CategoryID: categoryID, Color: color})
	return err
}

type skillIDInput struct {
	ID string `json:"id"`
}

// ArchiveSkill calls skill.archive.
func (c *Client) ArchiveSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.archive", skillIDInput{ID: id})
	return err
}

// UnarchiveSkill calls skill.unarchive.
func (c *Client) UnarchiveSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.unarchive", skillIDInput{ID: id})
	return err
}

// DeleteSkill calls skill.delete.
func (c *Client) DeleteSkill(id string) error {
	_, err := mutation[struct{}](c, "skill.delete", skillIDInput{ID: id})
	return err
}

// --- materialType queries ---

// GetAllMaterialTypes calls materialType.getAll.
func (c *Client) GetAllMaterialTypes() ([]model.MaterialType, error) {
	out, err := query[[]model.MaterialType](c, "materialType.getAll", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --- material mutations ---

type materialCreateInput struct {
	Name           string  `json:"name"`
	SkillID        string  `json:"skillId"`
	TypeID         string  `json:"typeId"`
	UnitType       string  `json:"unitType"`
	TotalUnits     float64 `json:"totalUnits"`
	URL            *string `json:"url,omitempty"`
	Notes          *string `json:"notes,omitempty"`
	StartDate      *string `json:"startDate,omitempty"`
	CompletedDate  *string `json:"completedDate,omitempty"`
	WeeklyUnitGoal *int    `json:"weeklyUnitGoal,omitempty"`
}

// MaterialCreateResult carries the fields needed to create a material.
// Populated by common.MaterialFormResult and passed to CreateMaterial.
type MaterialCreateResult struct {
	Name          string
	SkillID       string
	TypeID        string
	UnitType      string
	TotalUnits    float64
	URL           *string
	Notes         *string
	StartDate     *string
	CompletedDate *string
	WeeklyGoal    *int
}

// CreateMaterial calls material.create.
func (c *Client) CreateMaterial(r MaterialCreateResult) error {
	input := materialCreateInput{
		Name:           r.Name,
		SkillID:        r.SkillID,
		TypeID:         r.TypeID,
		UnitType:       r.UnitType,
		TotalUnits:     r.TotalUnits,
		URL:            r.URL,
		Notes:          r.Notes,
		WeeklyUnitGoal: r.WeeklyGoal,
	}
	// Date fields need SuperJSON meta so tRPC deserializes them as JS Dates.
	var datePaths []string
	if r.StartDate != nil {
		s := *r.StartDate + "T00:00:00.000Z"
		input.StartDate = &s
		datePaths = append(datePaths, "startDate")
	}
	if r.CompletedDate != nil {
		s := *r.CompletedDate + "T00:00:00.000Z"
		input.CompletedDate = &s
		datePaths = append(datePaths, "completedDate")
	}
	_, err := mutationWithMeta[struct{}](c, "material.create", superJSONDates(input, datePaths...))
	return err
}

type materialUpdateInput struct {
	ID             string   `json:"id"`
	Name           *string  `json:"name,omitempty"`
	SkillID        *string  `json:"skillId,omitempty"`
	TypeID         *string  `json:"typeId,omitempty"`
	UnitType       *string  `json:"unitType,omitempty"`
	TotalUnits     *float64 `json:"totalUnits,omitempty"`
	URL            *string  `json:"url,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	StartDate      *string  `json:"startDate,omitempty"`
	CompletedDate  *string  `json:"completedDate,omitempty"`
	WeeklyUnitGoal *int     `json:"weeklyUnitGoal,omitempty"`
}

// MaterialUpdateResult carries the fields needed to update a material.
type MaterialUpdateResult struct {
	ID            string
	Name          string
	SkillID       string
	TypeID        string
	UnitType      string
	TotalUnits    float64
	URL           *string
	Notes         *string
	StartDate     *string
	CompletedDate *string
	WeeklyGoal    *int
}

// UpdateMaterial calls material.update.
func (c *Client) UpdateMaterial(r MaterialUpdateResult) error {
	input := materialUpdateInput{
		ID:             r.ID,
		Name:           &r.Name,
		SkillID:        &r.SkillID,
		TypeID:         &r.TypeID,
		UnitType:       &r.UnitType,
		TotalUnits:     &r.TotalUnits,
		URL:            r.URL,
		Notes:          r.Notes,
		WeeklyUnitGoal: r.WeeklyGoal,
	}
	var datePaths []string
	if r.StartDate != nil {
		s := *r.StartDate + "T00:00:00.000Z"
		input.StartDate = &s
		datePaths = append(datePaths, "startDate")
	}
	if r.CompletedDate != nil {
		s := *r.CompletedDate + "T00:00:00.000Z"
		input.CompletedDate = &s
		datePaths = append(datePaths, "completedDate")
	}
	_, err := mutationWithMeta[struct{}](c, "material.update", superJSONDates(input, datePaths...))
	return err
}

type materialDeleteInput struct {
	ID string `json:"id"`
}

// DeleteMaterial calls material.delete.
func (c *Client) DeleteMaterial(id string) error {
	_, err := mutation[struct{}](c, "material.delete", materialDeleteInput{ID: id})
	return err
}

// --- exercise procedures ---

type exerciseCreateInput struct {
	Name string `json:"name"`
}

// CreateExercise calls exercise.create.
func (c *Client) CreateExercise(name string) error {
	_, err := mutation[struct{}](c, "exercise.create", exerciseCreateInput{Name: name})
	return err
}

type exerciseUpdateInput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// UpdateExercise calls exercise.update.
func (c *Client) UpdateExercise(id, name string) error {
	_, err := mutation[struct{}](c, "exercise.update", exerciseUpdateInput{ID: id, Name: name})
	return err
}

type exerciseIDInput struct {
	ID string `json:"id"`
}

// ArchiveExercise calls exercise.archive.
func (c *Client) ArchiveExercise(id string) error {
	_, err := mutation[struct{}](c, "exercise.archive", exerciseIDInput{ID: id})
	return err
}

// UnarchiveExercise calls exercise.unarchive.
func (c *Client) UnarchiveExercise(id string) error {
	_, err := mutation[struct{}](c, "exercise.unarchive", exerciseIDInput{ID: id})
	return err
}

// DeleteExercise calls exercise.delete.
func (c *Client) DeleteExercise(id string) error {
	_, err := mutation[struct{}](c, "exercise.delete", exerciseIDInput{ID: id})
	return err
}

type getAllExercisesInput struct {
	IncludeArchived bool `json:"includeArchived,omitempty"`
}

// GetAllExercises calls exercise.getAll.
func (c *Client) GetAllExercises(includeArchived bool) ([]model.Exercise, error) {
	out, err := query[[]model.Exercise](c, "exercise.getAll", getAllExercisesInput{IncludeArchived: includeArchived})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --- workout procedures ---

type getWorkoutSessionsInput struct {
	Type     *string `json:"type,omitempty"`
	DateFrom *string `json:"dateFrom,omitempty"`
	DateTo   *string `json:"dateTo,omitempty"`
	Limit    *int    `json:"limit,omitempty"`
}

// GetWorkoutSessions calls workout.getSessions.
// It also computes a Details string for each session from the embedded
// lift entries / run segments that the server already returns.
func (c *Client) GetWorkoutSessions(workoutType *string, limit *int) ([]model.WorkoutSession, error) {
	out, err := query[[]model.WorkoutSession](c, "workout.getSessions", getWorkoutSessionsInput{Type: workoutType, Limit: limit})
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].Details = formatSessionDetails(&out[i])
	}
	return out, nil
}

// formatSessionDetails builds a short human-readable summary of a session's
// exercises (lifting) or segments (running), formatted entirely in the client
// layer so TUI code stays clean.
func formatSessionDetails(s *model.WorkoutSession) string {
	if s.Type == model.WorkoutTypeLifting {
		parts := make([]string, 0, len(s.LiftEntries))
		for _, e := range s.LiftEntries {
			parts = append(parts, fmt.Sprintf("%s %.0f", e.Exercise.Name, e.WeightLbs))
		}
		return truncate(strings.Join(parts, ", "), 50)
	}
	// Running
	if s.RunEntry == nil || len(s.RunEntry.Segments) == 0 {
		return ""
	}
	segs := s.RunEntry.Segments
	if len(segs) == 1 {
		seg := segs[0]
		return fmt.Sprintf("%s %.2fmi %s", runZoneLabel(seg.Zone), seg.DistanceMiles, fmtSecs(seg.DurationSeconds))
	}
	parts := make([]string, 0, len(segs))
	for _, seg := range segs {
		parts = append(parts, fmt.Sprintf("%s %.1fmi", runZoneLabel(seg.Zone), seg.DistanceMiles))
	}
	return truncate(strings.Join(parts, ", "), 50)
}

// runZoneLabel maps a RunZone enum value to its human-readable label,
// matching the web app's RUN_ZONE_LABELS in src/lib/workout-utils.ts.
func runZoneLabel(zone string) string {
	switch zone {
	case "Z1":
		return "Zone 1"
	case "Z2":
		return "Zone 2"
	case "Z3":
		return "Zone 3"
	case "Z4":
		return "Zone 4"
	case "Z5":
		return "Zone 5"
	case "FREE":
		return "Free Run"
	default:
		return zone
	}
}

// fmtSecs formats a duration in seconds as "Xm Ys" or "Xh Ym".
func fmtSecs(secs int) string {
	if secs <= 0 {
		return "—"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// truncate shortens s to at most n runes, appending "…" if truncated.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

type getWorkoutSessionByIDInput struct {
	ID string `json:"sessionId"`
}

// GetWorkoutSessionByID calls workout.getSessionById.
func (c *Client) GetWorkoutSessionByID(id string) (*model.WorkoutSessionDetail, error) {
	out, err := query[model.WorkoutSessionDetail](c, "workout.getSessionById", getWorkoutSessionByIDInput{ID: id})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type liftEntryInput struct {
	ExerciseID string  `json:"exerciseId"`
	WeightLbs  float64 `json:"weightLbs"`
}

type logLiftSessionInput struct {
	Date            string           `json:"date"`
	DurationMinutes *int             `json:"durationMinutes,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	Lifts           []liftEntryInput `json:"lifts"`
}

// LogLiftSession calls workout.logLiftSession.
func (c *Client) LogLiftSession(date string, durationMinutes *int, notes *string, lifts []liftEntryInput) error {
	input := logLiftSessionInput{
		Date:            date + "T00:00:00.000Z",
		DurationMinutes: durationMinutes,
		Notes:           notes,
		Lifts:           lifts,
	}
	_, err := mutationWithMeta[struct{}](c, "workout.logLiftSession", superJSONDates(input, "date"))
	return err
}

// LiftEntry is an exported type for callers building lift session inputs.
type LiftEntry = liftEntryInput

type runSegmentInput struct {
	Zone            *string `json:"zone,omitempty"`
	DistanceMiles   float64 `json:"distanceMiles"`
	DurationSeconds int     `json:"durationSeconds"`
}

type logRunSessionInput struct {
	Date     string            `json:"date"`
	Notes    *string           `json:"notes,omitempty"`
	Segments []runSegmentInput `json:"segments"`
}

// RunSegment is an exported type for callers building run session inputs.
type RunSegment = runSegmentInput

// LogRunSession calls workout.logRunSession.
func (c *Client) LogRunSession(date string, notes *string, segments []runSegmentInput) error {
	input := logRunSessionInput{
		Date:     date + "T00:00:00.000Z",
		Notes:    notes,
		Segments: segments,
	}
	_, err := mutationWithMeta[struct{}](c, "workout.logRunSession", superJSONDates(input, "date"))
	return err
}

type workoutSessionIDInput struct {
	ID string `json:"sessionId"`
}

// DeleteWorkoutSession calls workout.deleteSession.
func (c *Client) DeleteWorkoutSession(id string) error {
	_, err := mutation[struct{}](c, "workout.deleteSession", workoutSessionIDInput{ID: id})
	return err
}

// GetWorkoutGoals calls workout.getGoals.
func (c *Client) GetWorkoutGoals() ([]model.WorkoutGoal, error) {
	out, err := query[[]model.WorkoutGoal](c, "workout.getGoals", struct{}{})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type setWorkoutGoalInput struct {
	Type            string `json:"type"`
	SessionsPerWeek int    `json:"sessionsPerWeek"`
}

// SetWorkoutGoal calls workout.setGoal.
func (c *Client) SetWorkoutGoal(workoutType string, sessionsPerWeek int) error {
	_, err := mutation[struct{}](c, "workout.setGoal", setWorkoutGoalInput{Type: workoutType, SessionsPerWeek: sessionsPerWeek})
	return err
}

// GetWorkoutStats calls workout.getStats.
func (c *Client) GetWorkoutStats() (*model.WorkoutStats, error) {
	out, err := query[model.WorkoutStats](c, "workout.getStats", struct{}{})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getLiftProgressInput struct {
	ExerciseID string `json:"exerciseId"`
	Days       *int   `json:"days,omitempty"`
}

// GetLiftProgress calls workout.getLiftProgress.
func (c *Client) GetLiftProgress(exerciseID string) (*model.LiftProgress, error) {
	out, err := query[model.LiftProgress](c, "workout.getLiftProgress", getLiftProgressInput{ExerciseID: exerciseID})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type getRunProgressInput struct {
	Days *int `json:"days,omitempty"`
}

// GetRunProgress calls workout.getRunProgress.
func (c *Client) GetRunProgress() (model.RunProgress, error) {
	return query[model.RunProgress](c, "workout.getRunProgress", getRunProgressInput{})
}

// --- workout update ---

// UpdateLiftSessionInput holds the fields needed to update a lifting session.
type UpdateLiftSessionInput struct {
	SessionID       string
	Date            *string // "YYYY-MM-DD", optional
	DurationMinutes *int    // optional; nil = clear
	Notes           *string // optional; nil = clear
	Lifts           []LiftEntry
}

// UpdateRunSessionInput holds the fields needed to update a running session.
type UpdateRunSessionInput struct {
	SessionID string
	Date      *string // "YYYY-MM-DD", optional
	Notes     *string // optional; nil = clear
	Segments  []RunSegment
}

type updateLiftSessionPayload struct {
	Type            string           `json:"type"`
	SessionID       string           `json:"sessionId"`
	Date            *string          `json:"date,omitempty"`
	DurationMinutes *int             `json:"durationMinutes,omitempty"`
	Notes           *string          `json:"notes,omitempty"`
	Lifts           []liftEntryInput `json:"lifts,omitempty"`
}

type updateRunSessionPayload struct {
	Type      string            `json:"type"`
	SessionID string            `json:"sessionId"`
	Date      *string           `json:"date,omitempty"`
	Notes     *string           `json:"notes,omitempty"`
	Segments  []runSegmentInput `json:"segments,omitempty"`
}

// UpdateLiftSession calls workout.updateSession for a LIFTING session.
func (c *Client) UpdateLiftSession(input UpdateLiftSessionInput) error {
	payload := updateLiftSessionPayload{
		Type:      "LIFTING",
		SessionID: input.SessionID,
		Notes:     input.Notes,
		Lifts:     input.Lifts,
	}
	var datePaths []string
	if input.Date != nil {
		s := *input.Date + "T00:00:00.000Z"
		payload.Date = &s
		datePaths = append(datePaths, "date")
	}
	if input.DurationMinutes != nil {
		payload.DurationMinutes = input.DurationMinutes
	}
	_, err := mutationWithMeta[struct{}](c, "workout.updateSession", superJSONDates(payload, datePaths...))
	return err
}

// UpdateRunSession calls workout.updateSession for a RUNNING session.
func (c *Client) UpdateRunSession(input UpdateRunSessionInput) error {
	payload := updateRunSessionPayload{
		Type:      "RUNNING",
		SessionID: input.SessionID,
		Notes:     input.Notes,
		Segments:  input.Segments,
	}
	var datePaths []string
	if input.Date != nil {
		s := *input.Date + "T00:00:00.000Z"
		payload.Date = &s
		datePaths = append(datePaths, "date")
	}
	_, err := mutationWithMeta[struct{}](c, "workout.updateSession", superJSONDates(payload, datePaths...))
	return err
}
