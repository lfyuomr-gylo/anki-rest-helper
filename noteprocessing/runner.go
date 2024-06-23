package noteprocessing

import (
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/util/execx"
	"anki-rest-enhancer/util/stringx"
	"context"
	"encoding/json"
	"github.com/joomcode/errorx"
	"log"
	"strings"
)

func NewScriptRunner() *scriptRunner {
	return &scriptRunner{}
}

type scriptRunner struct {
	// nop
}

var _ ScriptRunner = (*scriptRunner)(nil)

func (r *scriptRunner) RunScript(
	ctx context.Context,
	rule ankihelperconf.NoteProcessingRule,
	note NoteData,
	progress ProgressInfo,
) ([]Modification, error) {

	args, stdin, err := r.prepareArgsAndStdin(note, rule)
	if err != nil {
		return nil, err
	}

	cmdCtx := ctx
	if rule.Timeout > 0 {
		// currently this context deadline is not respected by
		ctx, cancel := context.WithTimeout(cmdCtx, rule.Timeout)
		defer cancel()
		cmdCtx = ctx
	}
	r.logRun(progress, rule.Exec.Command, args)
	cmdOut, err := execx.RunAndCollectOutput(cmdCtx, execx.Params{
		Command: rule.Exec.Command,
		Args:    args,
		Stdin:   stdin,
	})
	if err != nil {
		return nil, errorx.ExternalError.Wrap(err, "Note population command failed")
	}

	var commandOutParsed []Modification
	if !stringx.IsBlank(string(cmdOut)) {
		if err := json.Unmarshal(cmdOut, &commandOutParsed); err != nil {
			return nil, errorx.ExternalError.Wrap(err, "Note processing command's stdout is malformed")
		}
		for idx, modification := range commandOutParsed {
			if err := modification.Validate(); err != nil {
				return nil, errorx.Decorate(err, "Note processing command's stdout contains malformed modification #%d", idx)
			}
		}
	}

	return commandOutParsed, nil
}

func (r *scriptRunner) prepareArgsAndStdin(note NoteData, rule ankihelperconf.NoteProcessingRule) ([]string, string, error) {
	templateData := TemplateData{Note: note}

	args := make([]string, len(rule.Exec.Args))
	for i, arg := range rule.Exec.Args {
		switch {
		case arg.PlainString != nil:
			args[i] = *arg.PlainString
		case arg.Template != nil:
			var argBuilder strings.Builder
			if err := arg.Template.Execute(&argBuilder, templateData); err != nil {
				return nil, "", errorx.IllegalFormat.Wrap(err, "failed to substitute template in argument #%d", i)
			}
			args[i] = argBuilder.String()
		}
	}

	var stdin string
	switch {
	case rule.Exec.Stdin.PlainString != nil:
		stdin = *rule.Exec.Stdin.PlainString
	case rule.Exec.Stdin.Template != nil:
		var stdinBuilder strings.Builder
		if err := rule.Exec.Stdin.Template.Execute(&stdinBuilder, templateData); err != nil {
			return nil, "", errorx.IllegalFormat.Wrap(err, "failed to substitute template in stdin of the script")
		}
		stdin = stdinBuilder.String()
	}
	return args, stdin, nil
}

func (r *scriptRunner) logRun(progress ProgressInfo, command string, args []string) {
	log.Printf(
		"Executing note processing command [%d/%d]: %s '%s'",
		progress.CurrentNoteIndex,
		progress.TotalNotesCount,
		command,
		strings.Join(args, "' '"),
	)
}
