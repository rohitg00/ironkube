package engine

import "context"

type Hook interface {
	Name() string
}

type PrePhaseHook interface {
	Hook
	PrePhase(ctx context.Context, phaseName string, state *State) error
}

type PostPhaseHook interface {
	Hook
	PostPhase(ctx context.Context, phaseName string, state *State, phaseErr error) error
}

type ErrorHook interface {
	Hook
	OnError(ctx context.Context, phaseName string, state *State, err error) error
}
