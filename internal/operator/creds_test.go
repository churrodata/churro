package operator

import (
	"fmt"
	"testing"
	"time"
)

func TestGenerateChurroCreds(t *testing.T) {

	pipelineName := "pipeline1"
	serviceHosts := fmt.Sprintf("*.%s.svc.cluster.local,churro-watch.%s.svc.cluster.local,churro-ctl.%s.svc.cluster.local,localhost,churro-watch,churro-ctl,127.0.0.1", pipelineName, pipelineName, pipelineName)
	rsaBits := 4096
	validFor, err := time.ParseDuration("8760h")
	if err != nil {
		t.Fatalf("operator.GenerateChurroCreds Error: %v", err)
	}

	creds, err := GenerateChurroCreds(pipelineName, serviceHosts, rsaBits, validFor)
	if err != nil {
		t.Fatalf("operator.GenerateChurroCreds Error: %v", err)
	}
	if len(creds.Servicecrt) == 0 {
		t.Fatalf("operator.GenerateChurroCreds expected >0 length service.crt")
	}

}
