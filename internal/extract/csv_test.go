package extract

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"testing"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"
)

func TestExtractCSV(t *testing.T) {

	csvFileContent :=
		`
num,city,zip
2,boerne,78006
3,keller,76248
`

	// create a temp file based on the example data
	f, err := ioutil.TempFile("/tmp", "mycsvtest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.WriteString(csvFileContent)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

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
		Name:         "my-csv-files",
		Path:         "/tmp",
		Scheme:       extractapi.CSVScheme,
		ExtractRules: extractRules,
		Tablename:    "mycsvtable",
		Skipheaders:  1,
	}

	s := Server{
		DBCreds:       config.DBCredentials{},
		FileName:      f.Name(),
		Pi:            pipeline,
		ExtractSource: extractSource,
		SchemeValue:   extractapi.CSVScheme,
	}

	err = s.ExtractCSV(context.TODO())
	if err != nil {
		t.Fatalf("extract.ExtractCSV Error: %v", err)
	}

}
