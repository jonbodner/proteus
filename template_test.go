package proteus

import (
	"bytes"
	"fmt"
	"html/template"
	"testing"
)

func TestTemplate(t *testing.T) {
	tpl := addSlice("vals")

	funcMap := template.FuncMap{
		"join": joinFactory(1, Postgres),
	}

	tmpl, err := template.New("template_test").Funcs(funcMap).Parse(tpl)
	if err != nil {
		t.Error(err)
		return
	}
	var b bytes.Buffer

	err = tmpl.Execute(&b, map[string]interface{}{"vals": 3})
	if err != nil {
		t.Error(err)
	}
	fmt.Println("b:", b.String())
	if b.String() != "$1, $2, $3" {
		t.Errorf("Expected $1, $2, $3, got %s", b.String())
	}
}
