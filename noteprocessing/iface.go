package noteprocessing

import (
	"anki-rest-enhancer/ankihelperconf"
	"context"
)

type ProgressInfo struct {
	CurrentNoteIndex int
	TotalNotesCount  int
}

type ScriptRunner interface {
	// RunScript executes the script, providing it with the note data, and returns the note modification commands
	// produced by the script.
	RunScript(
		ctx context.Context,
		rule ankihelperconf.NoteProcessingRule,
		note NoteData,
		// for logging purposes
		progress ProgressInfo,
	) ([]Modification, error)
}
