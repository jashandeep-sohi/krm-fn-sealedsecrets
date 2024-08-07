package types

import (
	"encoding/json"
	"strconv"
)

type Bool bool

func (b *Bool) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	val, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}

	*b = Bool(val)
	return nil
}
