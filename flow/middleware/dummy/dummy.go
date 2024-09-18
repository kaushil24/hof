package dummy

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/hofstadter-io/hof/cmd/hof/flags"
	hofcontext "github.com/hofstadter-io/hof/flow/context"
)

type Dummy struct {
	val  cue.Value
	next hofcontext.Runner
}

func NewDummy(opts flags.RootPflagpole, popts flags.FlowPflagpole) *Dummy {
	return &Dummy{}
}

func (M *Dummy) Run(ctx *hofcontext.Context) (results interface{}, err error) {
	// Print important variables from the context
	fmt.Printf("Context: %+v\n", ctx)
	fmt.Printf("Current Path: %s\n", M.val.Path())

	// Modify cue.Value if it matches a condition (e.g., has a specific field)
	if field, err := M.val.LookupField("modifyMe"); err == nil {
		newVal := field.Value.FillPath(cue.ParsePath("newField"), "modified")
		M.val = M.val.FillPath(cue.ParsePath(field.Selector), newVal)
		fmt.Printf("Modified value: %v\n", M.val)
	}

	// Run the next middleware in the chain
	result, err := M.next.Run(ctx)

	fmt.Printf("Result: %+v\n", result)
	fmt.Printf("Error: %v\n", err)

	return result, err
}

func (M *Dummy) Apply(ctx *hofcontext.Context, runner hofcontext.RunnerFunc) hofcontext.RunnerFunc {
	return func(val cue.Value) (hofcontext.Runner, error) {
		hasAttr := false
		attrs := val.Attributes(cue.ValueAttr)
		for _, attr := range attrs {
			if attr.Name() == "dummy" {
				hasAttr = true
				break
			}
		}

		next, err := runner(val)
		if err != nil {
			return nil, err
		}

		if !hasAttr {
			return next, nil
		}

		fmt.Printf("Dummy middleware applied to: %s\n", val.Path())

		return &Dummy{
			val:  val,
			next: next,
		}, nil
	}
}
