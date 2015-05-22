package main

import (
	"fmt"
)

type ErrVariableNotSet struct {
	name string
}

func (err *ErrVariableNotSet) Error() string {
	return fmt.Sprintf("environment variable not set: %v", err.name)
}
