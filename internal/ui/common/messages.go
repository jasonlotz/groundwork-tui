package common

// GoBackMsg is sent by any screen to signal the root model to pop back.
type GoBackMsg struct{}

// ToastMsg carries a transient notification message to be shown by the root.
type ToastMsg struct {
	Text    string
	IsError bool
}

// ErrMsg wraps an error for use as a Bubble Tea message.
type ErrMsg struct{ Err error }

// --- Domain events ---
// These are broadcast globally by mutation sites. Any screen that cares about
// a domain change handles the event and reloads itself, regardless of whether
// it is currently active or in the background.

// MaterialChangedMsg is broadcast after any material is created, updated, or deleted.
type MaterialChangedMsg struct{}

// LearningLoggedMsg is broadcast after a learning log entry is successfully created.
type LearningLoggedMsg struct{}

// SkillChangedMsg is broadcast after any skill is created, updated, archived, unarchived, or deleted.
type SkillChangedMsg struct{}

// CategoryChangedMsg is broadcast after any category is created, updated, archived, unarchived, or deleted.
type CategoryChangedMsg struct{}

// ThemeChangedMsg is sent by the settings screen after the user picks a new theme.
// ThemeName is the name of the newly selected theme.
type ThemeChangedMsg struct{ ThemeName string }

// WorkoutLoggedMsg is broadcast after a workout session is successfully logged or deleted.
type WorkoutLoggedMsg struct{}

// ExerciseChangedMsg is broadcast after an exercise is created, updated, archived, or deleted.
type ExerciseChangedMsg struct{}

// SubtypeChangedMsg is broadcast after a workout subtype is created, updated, archived, or deleted.
type SubtypeChangedMsg struct{}

// HabitChangedMsg is broadcast after any habit is created, updated, toggled, or deleted.
type HabitChangedMsg struct{}
