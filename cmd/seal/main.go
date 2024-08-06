package main

import (
	"fmt"
	"os"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/seal"
	"github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/version"
)

func main() {
	if err := fn.AsMain(fn.ResourceListProcessorFunc(process)); err != nil {
		os.Exit(1)
	}
}

func process(rl *fn.ResourceList) (bool, error) {
	rl.Results.Infof(fmt.Sprintf("krm-fn-sealedsecets-seal (version=%s, url=%s)", version.Name, version.URL))

	return seal.Process(rl)
}
