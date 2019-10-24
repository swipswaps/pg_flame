package html

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"pg_flame/pkg/plan"
)

type Flame struct {
	Name     string  `json:"name"`
	Value    float64 `json:"value"`
	Time     float64 `json:"time"`
	Detail   string  `json:"detail"`
	Color    string  `json:"color"`
	InitPlan bool    `json:"init_plan"`
	Children []Flame `json:"children"`
}

const tableHeader = `<table class="table table-striped table-bordered"><tbody>`
const rowTemplate = "<tr><th>%s</th><td>%v</td></tr>"
const tableFooter = `</tbody></table>`

const detailTemplate = "<span>%s</span>"

const colorPlan = "#00C05A"
const colorInit = "#C0C0C0"

func Generate(w io.Writer, p plan.Plan) error {
	f := buildFlame(p)

	t, err := template.New("pg_flame").Parse(templateHTML)
	if err != nil {
		return err
	}

	flameJSON, err := json.Marshal(f)
	if err != nil {
		return err
	}

	data := struct {
		Data template.JS
	}{
		Data: template.JS(flameJSON),
	}

	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}

func buildFlame(p plan.Plan) Flame {
	planningFlame := Flame{
		Name:   "Query Planning",
		Value:  p.PlanningTime,
		Time:   p.PlanningTime,
		Detail: fmt.Sprintf(detailTemplate, "Time to generate the query plan"),
		Color:  colorPlan,
	}

	executionFlame := convertPlanNode(p.ExecutionTree, "")

	return Flame{
		Name:     "Total",
		Value:    planningFlame.Value + executionFlame.Value,
		Time:     planningFlame.Time + executionFlame.Time,
		Detail:   fmt.Sprintf(detailTemplate, "Includes planning and execution time"),
		Children: []Flame{planningFlame, executionFlame},
	}
}

func convertPlanNode(n plan.Node, color string) Flame {
	initPlan := n.ParentRelationship == "InitPlan"
	value := n.TotalTime

	if initPlan {
		color = colorInit
	}

	var childFlames []Flame
	for _, childNode := range n.Children {

		// Pass the color forward for grey InitPlan trees
		f := convertPlanNode(childNode, color)

		// Add to the total value if the child is an InitPlan node
		if f.InitPlan {
			value += f.Value
		}

		childFlames = append(childFlames, f)
	}

	return Flame{
		Name:     name(n),
		Value:    value,
		Time:     n.TotalTime,
		Detail:   detail(n),
		Color:    color,
		InitPlan: initPlan,
		Children: childFlames,
	}
}

func name(n plan.Node) string {
	switch {
	case n.Table != "" && n.Index != "":
		return fmt.Sprintf("%s using %s on %s", n.Method, n.Index, n.Table)
	case n.Table != "":
		return fmt.Sprintf("%s on %s", n.Method, n.Table)
	default:
		return n.Method
	}
}

func detail(n plan.Node) string {
	var b strings.Builder
	b.WriteString(tableHeader)

	if n.ParentRelationship != "" {
		fmt.Fprintf(&b, rowTemplate, "Parent Relationship", n.ParentRelationship)
	}

	if n.Filter != "" {
		fmt.Fprintf(&b, rowTemplate, "Filter", n.Filter)
	}

	if n.JoinFilter != "" {
		fmt.Fprintf(&b, rowTemplate, "Join Filter", n.JoinFilter)
	}

	if n.HashCond != "" {
		fmt.Fprintf(&b, rowTemplate, "Hash Cond", n.HashCond)
	}

	if n.IndexCond != "" {
		fmt.Fprintf(&b, rowTemplate, "Index Cond", n.IndexCond)
	}

	if n.RecheckCond != "" {
		fmt.Fprintf(&b, rowTemplate, "Recheck Cond", n.RecheckCond)
	}

	if n.BuffersHit != 0 {
		fmt.Fprintf(&b, rowTemplate, "Buffers Shared Hit", n.BuffersHit)
	}

	if n.BuffersRead != 0 {
		fmt.Fprintf(&b, rowTemplate, "Buffers Shared Read", n.BuffersRead)
	}

	if n.HashBuckets != 0 {
		fmt.Fprintf(&b, rowTemplate, "Hash Buckets", n.HashBuckets)
	}

	if n.HashBatches != 0 {
		fmt.Fprintf(&b, rowTemplate, "Hash Batches", n.HashBatches)
	}

	if n.MemoryUsage != 0 {
		fmt.Fprintf(&b, rowTemplate, "Memory Usage", fmt.Sprintf("%vkB", n.MemoryUsage))
	}

	b.WriteString(tableFooter)

	return b.String()
}
