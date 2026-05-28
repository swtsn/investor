// Package views contains the individual view models for the investor TUI.
package views

import "github.com/swtsn/investor/internal/domain"

// SharedState is the subset of app state that every view receives.
type SharedState struct {
	Buckets []domain.Bucket
	Width   int
	Height  int
}

// LoadMsg is sent to a view when it should (re-)fetch its data.
type LoadMsg struct {
	State SharedState
}

// BackMsg is emitted by a view when the user presses esc at the root step,
// signalling that the app should return to the Dashboard.
type BackMsg struct{}
