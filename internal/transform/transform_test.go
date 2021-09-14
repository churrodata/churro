package transform

import (
	"fmt"
	"testing"

	"github.com/churrodata/churro/internal/domain"
)

func TestRunRules(t *testing.T) {

	jkajdskfjadfa

	cols := []string{"one", "two"}
	record := []interface{}{"a", "b"}
	rules := make(map[string]domain.ExtractRule)
	r := domain.ExtractRule{
		ID:                "one",
		ExtractSourceID:   "something",
		ColumnName:        "one",
		ColumnPath:        "1",
		ColumnType:        "TEXT",
		TransformFunction: "transforms.MyUppercase",
	}
	rules[r.ID] = r
	functions := []domain.TransformFunction{
		{
			ID:   "one",
			Name: "transforms.MyUppercase",
			Source: `package transforms

      import "strings"

      func MyUppercase(s string) string {

        return strings.ToUpper(s)

      }`,
		},
	}
	err := RunRules(cols, record, rules, functions)
	if err != nil {
		t.Fatalf("transform.RunRules Error: %v", err)
	}
	fmt.Printf("%+v\n", record)
	if record[0] != "A" {
		t.Fatalf("transform.RunRules Expected 'A': %v", err)
	}

}
