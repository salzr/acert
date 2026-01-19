package bootstrap

import (
	"fmt"
	"testing"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
)

func TestFillOutPrompt(t *testing.T) {
	i := fillOutPrompt(&certmanagerv1.X509Subject{}).(*certmanagerv1.X509Subject)
	fmt.Printf("%+v\n", i)
}
