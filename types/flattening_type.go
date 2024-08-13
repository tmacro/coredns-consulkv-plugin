package types

import (
	"encoding/json"
	"fmt"
)

type FlatteningType string

const (
	Flattening_None  FlatteningType = "none"
	Flattening_Local FlatteningType = "local"
	Flattening_Full  FlatteningType = "full"
)

func (f FlatteningType) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(f))
}

func (f *FlatteningType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	switch FlatteningType(s) {
	case Flattening_None, Flattening_Local, Flattening_Full:
		*f = FlatteningType(s)
		return nil

	default:
		return fmt.Errorf("invalid FlatteningType: %s", s)
	}
}
