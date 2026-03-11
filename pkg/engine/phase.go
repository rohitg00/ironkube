package engine

import "context"

type Phase interface {
	Name() string
	Run(ctx context.Context, state *State) error
}

type CleanupPhase interface {
	Phase
	Cleanup(ctx context.Context, state *State) error
}
