package extract

import (
	"context"
	"testing"

	extractapi "github.com/churrodata/churro/api/extract"
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"
)

func TestExtractAPI(t *testing.T) {

	pipeline := v1alpha1.Pipeline{
		Spec: v1alpha1.PipelineSpec{
			DatabaseType: domain.DatabaseMock,
		},
	}

	shipNameRule := domain.ExtractRule{
		ID:              "rule1",
		ExtractSourceID: "one",
		ColumnName:      "shipname",
		ColumnPath:      "$[*].spaceTrack.OBJECT_NAME",
		ColumnType:      "TEXT",
	}
	longitudeRule := domain.ExtractRule{
		ID:              "rule2",
		ExtractSourceID: "two",
		ColumnName:      "longitude",
		ColumnPath:      "$[*].longitude",
		ColumnType:      "DECIMAL",
	}
	latitudeRule := domain.ExtractRule{
		ID:              "rule3",
		ExtractSourceID: "three",
		ColumnName:      "latitude",
		ColumnPath:      "$[*].latitude",
		ColumnType:      "DECIMAL",
	}

	extractRules := make(map[string]domain.ExtractRule)
	extractRules[shipNameRule.ID] = shipNameRule
	extractRules[latitudeRule.ID] = latitudeRule
	extractRules[longitudeRule.ID] = longitudeRule

	extractSource := domain.ExtractSource{
		ID:             "one",
		Name:           "my-starlink-api",
		Path:           "https://api.spacexdata.com/v4/starlink",
		Scheme:         extractapi.APIScheme,
		ExtractRules:   extractRules,
		Tablename:      "mystarlinktable",
		Cronexpression: "@every 3s",
	}

	s := Server{
		DBCreds:       config.DBCredentials{},
		Pi:            pipeline,
		ExtractSource: extractSource,
		SchemeValue:   extractapi.APIScheme,
		APIStopTime:   5,
	}

	err := s.ExtractAPI(context.TODO())
	if err != nil {
		t.Fatalf("extract.ExtractAPI Error: %v", err)
	}

}
