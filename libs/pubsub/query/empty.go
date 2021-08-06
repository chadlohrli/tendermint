package query

import (
	"github.com/providenetwork/tendermint/abci/types"
)

// Empty query matches any set of events.
type Empty struct {
}

// Matches always returns true.
func (Empty) Matches(events []types.Event) (bool, error) {
	return true, nil
}

func (Empty) String() string {
	return "empty"
}
