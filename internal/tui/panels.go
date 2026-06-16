package tui

// The three panels are concrete value types updated in the Elm/Bubble Tea
// style: every mutator returns an updated copy rather than mutating in place, so
// RootModel.Update stays a pure transformation. Each panel has its own file:
// NavPanel in nav.go, WorkspacePanel in workspace.go, LogPanel in log.go.
