package flow

import (
	"fmt"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/format"
	"cuelang.org/go/cue/parser"
	cueflow "cuelang.org/go/tools/flow"

	flowctx "github.com/hofstadter-io/hof/flow/context"
	"github.com/hofstadter-io/hof/flow/tasker"
	"github.com/hofstadter-io/hof/lib/cuetils"
	"github.com/hofstadter-io/hof/lib/hof"
)

type Flow struct {
	*hof.Node[Flow]

	Root       cue.Value
	Orig       cue.Value
	Final      cue.Value
	cueContext cue.Context

	FlowCtx *flowctx.Context
	Ctrl    *cueflow.Controller

	TaskOutputs map[string]cue.Value
	GlobalVars  map[string]cue.Value
}

func NewFlow(node *hof.Node[Flow], cctx *cue.Context) *Flow {
	return &Flow{
		Node:        node,
		Root:        node.Value,
		Orig:        node.Value,
		cueContext:  *cctx,
		TaskOutputs: make(map[string]cue.Value),
		GlobalVars:  make(map[string]cue.Value),
	}
}

func OldFlow(ctx *flowctx.Context, val cue.Value, cctx *cue.Context) (*Flow, error) {
	p := &Flow{
		Root:        val,
		Orig:        val,
		FlowCtx:     ctx,
		cueContext:  *cctx,
		TaskOutputs: make(map[string]cue.Value),
		GlobalVars:  make(map[string]cue.Value),
	}
	return p, nil
}

// This is for the top-level flows
func (P *Flow) Start() error {
	err := P.run()
	// fmt.Println("Start().Err", P.Orig.Path(), err)
	return err
}

func (P *Flow) run() error {
	if P.Node == nil {
		node, err := hof.ParseHof[Flow](P.Orig)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("Root flow value is not a flow, has nil #hof node")
		}
		P.Node = node
	}

	root := P.Root

	cfg := &cueflow.Config{
		IgnoreConcrete:  true,
		FindHiddenTasks: true,
		UpdateFunc: func(c *cueflow.Controller, t *cueflow.Task) error {
			if t != nil {
				v := t.Value()

				node, err := hof.ParseHof[any](v)
				if err != nil {
					return err
				}
				if node == nil {
					panic("we should have found a node to even get here")
				}

				if node.Hof.Flow.Task == "" {
					return nil
				}

				// Inject input variables before task execution
				err = P.injectInputVariables(t)
				if err != nil {
					return err
				}

				// Store task output after execution
				P.storeTaskOutput(t)

				if node.Hof.Flow.Print.Level > 0 && !node.Hof.Flow.Print.Before {
					pv := v.LookupPath(cue.ParsePath(node.Hof.Flow.Print.Path))
					if node.Hof.Path == "" {
						fmt.Printf("%s", node.Hof.Flow.Print.Path)
					} else if node.Hof.Flow.Print.Path == "" {
						fmt.Printf("%s", node.Hof.Path)
					} else {
						fmt.Printf("%s.%s", node.Hof.Path, node.Hof.Flow.Print.Path)
					}
					fmt.Printf(": %v\n", pv)
				}
			}
			return nil
		},
	}

	if P.Orig != P.Root {
		cfg.Root = P.Orig.Path()
	}

	v := P.Orig.Context().CompileString("{...}")
	u := v.Unify(root)

	P.Ctrl = cueflow.New(cfg, u, tasker.NewTasker(P.FlowCtx))

	err := P.Ctrl.Run(P.FlowCtx.GoContext)

	P.Final = P.Ctrl.Value()
	if err != nil {
		s := cuetils.CueErrorToString(err)
		return fmt.Errorf("Error in %s | %s: %s", P.Hof.Metadata.Name, P.Orig.Path(), s)
	}

	return nil
}

func (P *Flow) injectInputVariables(t *cueflow.Task) error {
	taskPath := t.Path().String()
	inputPath := cue.ParsePath(taskPath + ".input")

	// Get the source of the task value
	src, err := format.Node(t.Value().Syntax())
	if err != nil {
		return fmt.Errorf("error getting task source: %v", err)
	}

	// Parse the source into an AST
	f, err := parser.ParseFile("", src, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("error parsing task source: %v", err)
	}

	// Define a function to process @runinject attributes
	var processNode func(n ast.Node) bool
	processNode = func(n ast.Node) bool {
		if field, ok := n.(*ast.Field); ok {
			for _, attr := range field.Attrs {
				if strings.HasPrefix(attr.Text, "@runinject") {
					varName := parseRunInjectAttr(attr.Text)
					var value cue.Value

					// Check if it's a reference to a previous task's output
					if strings.Contains(varName, ".") {
						value = P.resolveTaskOutputReference(varName)
					} else {
						// Otherwise, treat it as a global variable
						value = P.GlobalVars[varName]
					}

					if !value.Exists() {
						fmt.Printf("Warning: No value found for @runinject(%s)\n", varName)
						return true
					}

					// Construct the full path
					var fieldName string
					switch label := field.Label.(type) {
					case *ast.Ident:
						fieldName = label.Name
					case *ast.BasicLit:
						fieldName = label.Value
					default:
						fmt.Printf("Unsupported label type: %T\n", field.Label)
						return true
					}

					fullPath := cue.ParsePath(inputPath.String() + "." + fieldName)

					// Use the Fill method to inject the value
					err := t.Fill(map[string]cue.Value{fullPath.String(): value})
					if err != nil {
						fmt.Printf("Error injecting value for %s: %v\n", varName, err)
					}
				}
			}
		}
		return true
	}

	// Walk the AST
	ast.Walk(f, processNode, nil)

	return nil
}

func (P *Flow) resolveTaskOutputReference(ref string) cue.Value {
	parts := strings.Split(ref, ".")
	if len(parts) < 2 {
		return cue.Value{}
	}

	taskName := parts[0]
	outputPath := strings.Join(parts[1:], ".")

	if output, ok := P.TaskOutputs[taskName]; ok {
		return output.LookupPath(cue.ParsePath(outputPath))
	}

	return cue.Value{}
}

func (P *Flow) storeTaskOutput(t *cueflow.Task) {
	taskPath := t.Path().String()
	outputPath := cue.ParsePath(taskPath + ".output")
	output := t.Value().LookupPath(outputPath)
	if output.Exists() {
		P.TaskOutputs[taskPath] = output
	}
}

func parseRunInjectAttr(attrText string) string {
	attrText = strings.TrimPrefix(attrText, "@runinject(")
	attrText = strings.TrimSuffix(attrText, ")")
	return strings.Trim(attrText, "\"")
}
