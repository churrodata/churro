package dataprov

import (
	"github.com/churrodata/churro/api/v1alpha1"
	"github.com/churrodata/churro/internal/domain"
	"github.com/churrodata/churro/pkg/config"

	"testing"
)

func TestRegister(t *testing.T) {

	dp := &domain.DataProvenance{}
	pipeline := v1alpha1.Pipeline{
		Spec: v1alpha1.PipelineSpec{
			DatabaseType: domain.DatabaseMock,
		},
	}
	dbcreds := config.DBCredentials{}
	err := Register(dp, pipeline, dbcreds)
	if err != nil {
		t.Fatalf("dataprov.Register Error: %v", err)
	}

}
