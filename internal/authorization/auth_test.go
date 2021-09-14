package authorization

import (
	"fmt"

	"github.com/churrodata/churro/internal/domain"

	"testing"
)

func TestAuthorized(t *testing.T) {

	m := AuthMap{
		ID:      "someuser",
		Subject: "admin@admin.org",
		Action:  ActionAdmin,
	}
	authOK := m.Authorized(domain.DatabaseMock)
	fmt.Printf("%v authOK\n", authOK)
	if authOK == true {
		t.Fatalf("auth.Authorized Error: %v", fmt.Errorf("expected authOK to be false"))
	}

}
