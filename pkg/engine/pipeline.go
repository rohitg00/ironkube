package engine

import "context"

type Pipeline struct {
	phases          []Phase
	hooks           []Hook
	state           *State
	completedPhases []string
}

func NewPipeline(phases ...Phase) *Pipeline {
	return &Pipeline{
		phases: phases,
		state:  NewState(),
	}
}

func (p *Pipeline) WithHooks(hooks ...Hook) *Pipeline {
	p.hooks = append(p.hooks, hooks...)
	return p
}

func (p *Pipeline) Execute(ctx context.Context) error {
	for _, phase := range p.phases {
		select {
		case <-ctx.Done():
			p.runCleanups(ctx)
			return ctx.Err()
		default:
		}

		if err := p.runPrePhaseHooks(ctx, phase.Name()); err != nil {
			p.runCleanups(ctx)
			return err
		}

		phaseErr := phase.Run(ctx, p.state)

		p.runPostPhaseHooks(ctx, phase.Name(), phaseErr)

		if phaseErr != nil {
			p.runErrorHooks(ctx, phase.Name(), phaseErr)
			p.runCleanups(ctx)
			return phaseErr
		}

		p.completedPhases = append(p.completedPhases, phase.Name())
	}
	return nil
}

func (p *Pipeline) runPrePhaseHooks(ctx context.Context, phaseName string) error {
	for _, h := range p.hooks {
		if pre, ok := h.(PrePhaseHook); ok {
			if err := pre.PrePhase(ctx, phaseName, p.state); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Pipeline) runPostPhaseHooks(ctx context.Context, phaseName string, phaseErr error) {
	for _, h := range p.hooks {
		if post, ok := h.(PostPhaseHook); ok {
			_ = post.PostPhase(ctx, phaseName, p.state, phaseErr)
		}
	}
}

func (p *Pipeline) runErrorHooks(ctx context.Context, phaseName string, err error) {
	for _, h := range p.hooks {
		if errHook, ok := h.(ErrorHook); ok {
			_ = errHook.OnError(ctx, phaseName, p.state, err)
		}
	}
}

func (p *Pipeline) runCleanups(ctx context.Context) {
	for i := len(p.completedPhases) - 1; i >= 0; i-- {
		name := p.completedPhases[i]
		for _, phase := range p.phases {
			if cp, ok := phase.(CleanupPhase); ok && phase.Name() == name {
				_ = cp.Cleanup(ctx, p.state)
			}
		}
	}
}

func (p *Pipeline) CompletedPhases() []string {
	return p.completedPhases
}

func (p *Pipeline) State() *State {
	return p.state
}
