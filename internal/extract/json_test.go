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

func TestExtractJSON(t *testing.T) {

	jsonFileContent :=
		`
{ "store": {
    "book": [
      { "category": "reference",
        "author": "Nigel Rees",
        "title": "Sayings of the Century",
        "price": 8.95
      },
      { "category": "fiction",
        "author": "Evelyn Waugh",
        "title": "Sword of Honour",
        "price": 12.99
      },
      { "category": "fiction",
        "author": "Herman Melville",
        "title": "Moby Dick",
        "isbn": "0-553-21311-3",
        "price": 8.99
      },
      { "category": "fiction",
        "author": "J. R. R. Tolkien",
        "title": "The Lord of the Rings",
        "isbn": "0-395-19395-8",
        "price": 22.99
      }
    ],
    "bicycle": {
      "color": "red",
      "price": 19.95
    }
  }
}
`

	// create a temp file based on the example data
	f, err := ioutil.TempFile("/tmp", "myjsontest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.WriteString(jsonFileContent)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	pipeline := v1alpha1.Pipeline{
		Spec: v1alpha1.PipelineSpec{
			DatabaseType: domain.DatabaseMock,
		},
	}

	extractRules := make(map[string]domain.ExtractRule)

	extractSource := domain.ExtractSource{
		ID:           "one",
		Name:         "my-json-files",
		Path:         "/tmp",
		Scheme:       extractapi.JSONScheme,
		ExtractRules: extractRules,
		Tablename:    "myjsontable",
	}

	s := Server{
		DBCreds:       config.DBCredentials{},
		FileName:      f.Name(),
		Pi:            pipeline,
		ExtractSource: extractSource,
		SchemeValue:   extractapi.JSONScheme,
	}

	err = s.ExtractJSON(context.TODO())
	if err != nil {
		t.Fatalf("extract.ExtractJSON Error: %v", err)
	}

}
