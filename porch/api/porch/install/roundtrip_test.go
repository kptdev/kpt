package install

import (
	"testing"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/fuzzer"
	"k8s.io/apimachinery/pkg/api/apitesting/roundtrip"
)

func TestRoundTripTypes(t *testing.T) {
	roundtrip.RoundTripTestForAPIGroup(t, Install, fuzzer.Funcs)
}
