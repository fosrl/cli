package companion

import (
	"context"
	"fmt"
)

// GuardReady returns an error when companion mode is enabled but the desktop client session is not ready.
func GuardReady(ctx context.Context) error {
	state := StateFromContext(ctx)
	if !state.Enabled || state.Active {
		return nil
	}
	return NotReadyError(state.ProviderName, state.NotReadySuggestion)
}

// NotReadyError formats an error for companion-enabled commands when the desktop client is unavailable.
func NotReadyError(clientName, suggestion string) error {
	if clientName == "" {
		clientName = "desktop app"
	}

	msg := fmt.Sprintf("Companion mode is enabled but the %s is not ready to use", clientName)
	if suggestion != "" {
		msg += "\n" + suggestion
	}
	return fmt.Errorf("%s", msg)
}

type stateCtxKeyType string

const stateCtxKey stateCtxKeyType = "companionState"

func WithState(ctx context.Context, state *State) context.Context {
	if state == nil {
		state = &State{}
	}
	return context.WithValue(ctx, stateCtxKey, state)
}

func StateFromContext(ctx context.Context) *State {
	state, ok := ctx.Value(stateCtxKey).(*State)
	if !ok || state == nil {
		return &State{}
	}
	return state
}

// GuardMutatingAuth returns an error when companion mode is enabled on this platform.
func GuardMutatingAuth(ctx context.Context) error {
	state := StateFromContext(ctx)
	if !state.Enabled {
		return nil
	}
	return MutatingAuthError(state.ProviderName)
}
