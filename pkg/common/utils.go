package common

import (
	"strconv"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

func NormalizeIndexAnnotation(rl *fn.ResourceList) error {
	indexByPath := make(map[string]int)

	for _, item := range rl.Items {
		path := item.PathAnnotation()
		if path == "" {
			continue
		}

		i, ok := indexByPath[path]
		if !ok {
			indexByPath[path] = 0
			i = 0
		} else {
			i++
			indexByPath[path] = i
		}

		err := item.SetAnnotation(fn.IndexAnnotation, strconv.Itoa(i))
		if err != nil {
			return err
		}
	}

	return nil
}
