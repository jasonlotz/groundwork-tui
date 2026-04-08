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
