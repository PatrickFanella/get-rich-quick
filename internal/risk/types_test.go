package risk

import "testing"

func TestCircuitBreakerStates(t *testing.T) {
	t.Parallel()

	if CircuitBreakerStateOpen != "open" {
		t.Fatalf("expected open state value, got %q", CircuitBreakerStateOpen)
	}
	if CircuitBreakerStateTripped != "tripped" {
		t.Fatalf("expected tripped state value, got %q", CircuitBreakerStateTripped)
	}
	if CircuitBreakerStateCooldown != "cooldown" {
		t.Fatalf("expected cooldown state value, got %q", CircuitBreakerStateCooldown)
	}

	states := map[CircuitBreakerState]struct{}{
		CircuitBreakerStateOpen:     {},
		CircuitBreakerStateTripped:  {},
		CircuitBreakerStateCooldown: {},
	}
	if len(states) != 3 {
		t.Fatalf("expected exactly 3 circuit breaker states, got %d", len(states))
	}
}

func TestKillSwitchMechanisms(t *testing.T) {
	t.Parallel()

	if KillSwitchMechanismAPI != "api_toggle" {
		t.Fatalf("expected api mechanism value, got %q", KillSwitchMechanismAPI)
	}
	if KillSwitchMechanismFile != "file_flag" {
		t.Fatalf("expected file mechanism value, got %q", KillSwitchMechanismFile)
	}
	if KillSwitchMechanismEnvVar != "env_var" {
		t.Fatalf("expected env mechanism value, got %q", KillSwitchMechanismEnvVar)
	}
	if KillSwitchMechanismUnknown != "unknown" {
		t.Fatalf("expected unknown mechanism value, got %q", KillSwitchMechanismUnknown)
	}

	mechanisms := map[KillSwitchMechanism]struct{}{
		KillSwitchMechanismAPI:     {},
		KillSwitchMechanismFile:    {},
		KillSwitchMechanismEnvVar:  {},
		KillSwitchMechanismUnknown: {},
	}
	if len(mechanisms) != 4 {
		t.Fatalf("expected exactly 4 kill switch mechanisms, got %d", len(mechanisms))
	}
}
