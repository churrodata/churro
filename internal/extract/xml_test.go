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

func TestExtractXML(t *testing.T) {

	xmlFileContent :=
		`
<library>
  <!-- Great book. -->
  <book id="b0836217462" available="true">
    <isbn>1</isbn>
    <title lang="en">Being a Dog Is a Full-Time Job</title>
    <quote>I'd dog paddle the deepest ocean.</quote>
    <author id="CMS">
      <?echo "go rocks"?>
      <name>Charles M Schulz</name>
      <born>1922-11-26</born>
      <dead>2000-02-12</dead>
    </author>
    <character id="PP">
      <name>Peppermint Patty</name>
      <born>1966-08-22</born>
      <qualification>bold, brash and tomboyish</qualification>
    </character>
    <character id="Snoopy">
      <name>Snoopy</name>
      <born>1950-10-04</born>
      <qualification>extroverted beagle</qualification>
    </character>
  </book>
</library>
<library>
  <!-- Great book. -->
  <book id="b0836217462" available="true">
    <isbn>2</isbn>
    <title lang="en">Being a Dog Is a Full-Time Job</title>
    <quote>I'd dog paddle the deepest ocean.</quote>
    <author id="CMS">
      <?echo "go rocks"?>
      <name>name2</name>
      <born>a-11-26</born>
      <dead>a-02-12</dead>
    </author>
    <character id="PP">
      <name>a Patty</name>
      <born>a-08-22</born>
      <qualification>bold, brash and tomboyish</qualification>
    </character>
    <character id="Snoopy">
      <name>a</name>
      <born>a-10-04</born>
      <qualification>extroverted beagle</qualification>
    </character>
  </book>
</library>
<library>
  <!-- Great book. -->
  <book id="b0836217462" available="true">
    <isbn>3</isbn>
    <title lang="en">Being a Dog Is a Full-Time Job</title>
    <quote>I'd dog paddle the deepest ocean.</quote>
    <author id="CMS">
      <?echo "go rocks"?>
      <name>name3</name>
      <born>a-11-26</born>
      <dead>a-02-12</dead>
    </author>
    <character id="PP">
      <name>a Patty</name>
      <born>a-08-22</born>
      <qualification>bold, brash and tomboyish</qualification>
    </character>
    <character id="Snoopy">
      <name>a</name>
      <born>a-10-04</born>
      <qualification>extroverted beagle</qualification>
    </character>
  </book>
</library>
	`

	// create a temp file based on the example data
	f, err := ioutil.TempFile("/tmp", "myxmltest")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, err = f.WriteString(xmlFileContent)
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
		ColumnName:      "author",
		ColumnPath:      "/library/book/author/name",
		ColumnType:      "TEXT",
	}

	extractRules := make(map[string]domain.ExtractRule)
	extractRules[rule.ID] = rule

	extractSource := domain.ExtractSource{
		ID:           "one",
		Name:         "my-xml-files",
		Path:         "/tmp",
		Scheme:       extractapi.XMLScheme,
		ExtractRules: extractRules,
		Tablename:    "myxmltable",
	}

	s := Server{
		DBCreds:       config.DBCredentials{},
		FileName:      f.Name(),
		Pi:            pipeline,
		ExtractSource: extractSource,
		SchemeValue:   extractapi.XMLScheme,
	}

	err = s.ExtractXML(context.TODO())
	if err != nil {
		t.Fatalf("extract.ExtractXML Error: %v", err)
	}

}
