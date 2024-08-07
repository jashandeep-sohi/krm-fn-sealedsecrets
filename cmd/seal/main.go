package main

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/seal"
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/version"
)

func main() {
	fn.Logf(fmt.Sprintf("krm-fn-sealedsecets-seal (version=%s, url=%s)", version.Name, version.URL))

	if err := fn.AsMain(fn.ResourceListProcessorFunc(seal.Process)); err != nil {
		os.Exit(1)
	}
}
