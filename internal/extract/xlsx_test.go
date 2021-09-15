package extract

import (
	"context"
	"testing"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"
)

func TestExtractXLSX(t *testing.T) {

	pipeline := v1alpha1.Pipeline{
		Spec: v1alpha1.PipelineSpec{
			DatabaseType: domain.DatabaseMock,
		},
	}

	rule := domain.ExtractRule{
		ID:              "rule1",
		ExtractSourceID: "one",
		ColumnName:      "city",
		ColumnPath:      "1",
		ColumnType:      "TEXT",
	}

	extractRules := make(map[string]domain.ExtractRule)
	extractRules[rule.ID] = rule

	extractSource := domain.ExtractSource{
		ID:           "one",
		Name:         "my-xlsx-files",
		Path:         "/tmp",
		Scheme:       extractapi.XLSXScheme,
		ExtractRules: extractRules,
		Tablename:    "myxlsxtable",
		Skipheaders:  1,
		Sheetname:    "Sheet1",
	}

	s := Server{
		DBCreds:       config.DBCredentials{},
		FileName:      "./Book1.xlsx",
		Pi:            pipeline,
		ExtractSource: extractSource,
		SchemeValue:   extractapi.XLSXScheme,
	}

	err := s.ExtractXLS(context.TODO())
	if err != nil {
		t.Fatalf("extract.ExtractXLS Error: %v", err)
	}

}
