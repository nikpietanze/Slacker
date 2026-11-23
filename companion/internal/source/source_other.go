//go:build !windows && !darwin && !linux

package source

import (
	"context"
	"errors"
)

type unsupported struct{}

// New returns the platform source.
func New(Options) Source { return unsupported{} }

func (unsupported) Name() string { return "unsupported OS" }
func (unsupported) Run(context.Context, func(int)) error {
	return errors.New("this operating system is not supported yet")
}
func (unsupported) Diagnose(context.Context) string { return "unsupported OS\n" }
