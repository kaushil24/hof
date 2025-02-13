# files copied from outside the repo
extern: cmdhelp schemas

schemas:
	# todo, get from github @ version
	rsync -ra ../schema/* code/hof-schemas
	rsync -a --include='*.cue' --include='*/' --exclude='*' ../flow/tasks/* code/hof-schemas/flow/tasks

.PHONY: cmdhelp
cmdhelp:
	@hof         -h > code/cmd-help/hof
	@hof mod     -h > code/cmd-help/mod
	@hof flow    -h > code/cmd-help/flow
	@hof dm      -h > code/cmd-help/dm
	@hof gen     -h > code/cmd-help/gen
	@hof fmt     -h > code/cmd-help/fmt
	@hof create  -h > code/cmd-help/create
	@hof chat    -h > code/cmd-help/chat
	@hof tui     -h > code/cmd-help/tui
	@hof eval    -h > code/cmd-help/eval
	@hof export  -h > code/cmd-help/export
	@hof vet     -h > code/cmd-help/vet
	@hof def     -h > code/cmd-help/def

