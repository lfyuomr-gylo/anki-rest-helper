package noteprocessing

import (
	"anki-rest-enhancer/ankihelperconf"
	"anki-rest-enhancer/util/execx"
	"anki-rest-enhancer/util/stringx"
	"context"
	"encoding/json"
	"github.com/joomcode/errorx"
	"log"
	"os"
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

	cmdCtx := ctx
	if rule.Timeout > 0 {
		// currently this context deadline is not respected by
		ctx, cancel := context.WithTimeout(cmdCtx, rule.Timeout)
		defer cancel()
		cmdCtx = ctx
	}

	params, err := r.prepareExecParams(note, rule)
	if err != nil {
		return nil, err
	}
	r.logRun(params, progress)
	cmdOut, err := execx.RunAndCollectOutput(cmdCtx, params)
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

func (r *scriptRunner) prepareExecParams(note NoteData, rule ankihelperconf.NoteProcessingRule) (execx.Params, error) {
	templateData := TemplateData{Note: note}

	args := make([]string, len(rule.Exec.Args))
	for i, arg := range rule.Exec.Args {
		switch {
		case arg.PlainString != nil:
			args[i] = *arg.PlainString
		case arg.Template != nil:
			var argBuilder strings.Builder
			if err := arg.Template.Execute(&argBuilder, templateData); err != nil {
				return execx.Params{}, errorx.IllegalFormat.Wrap(err, "failed to substitute template in argument #%d", i)
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
			return execx.Params{}, errorx.IllegalFormat.Wrap(err, "failed to substitute template in stdin of the script")
		}
		stdin = stdinBuilder.String()
	}

	var env []string
	if len(rule.Exec.Env) > 0 {
		extraEnv := make([]string, 0, len(rule.Exec.Env))
		for key, val := range rule.Exec.Env {
			extraEnv = append(extraEnv, key+"="+val)
		}
		env = append(os.Environ(), extraEnv...)
	}

	return execx.Params{
		Command: rule.Exec.Command,
		Args:    args,
		Stdin:   stdin,
		Env:     env,
	}, nil
}

func (r *scriptRunner) logRun(params execx.Params, progress ProgressInfo) {
	log.Printf(
		"Executing note processing command [%d/%d]: %s '%s'",
		progress.CurrentNoteIndex,
		progress.TotalNotesCount,
		params.Command,
		strings.Join(params.Args, "' '"),
	)
}
