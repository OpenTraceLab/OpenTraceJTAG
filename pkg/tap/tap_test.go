package tap

import "testing"

func TestNextStateTable(t *testing.T) {
	type transition struct {
		start State
		tms   bool
		end   State
	}

	cases := []transition{
		{StateTestLogicReset, false, StateRunTestIdle},
		{StateTestLogicReset, true, StateTestLogicReset},
		{StateRunTestIdle, true, StateSelectDRScan},
		{StateSelectDRScan, false, StateCaptureDR},
		{StateShiftDR, true, StateExit1DR},
		{StateExit2DR, false, StateShiftDR},
		{StateSelectIRScan, true, StateTestLogicReset},
		{StateCaptureIR, false, StateShiftIR},
		{StatePauseIR, true, StateExit2IR},
		{StateExit2IR, true, StateUpdateIR},
	}

	for _, tc := range cases {
		got := NextState(tc.start, tc.tms)
		if got != tc.end {
			t.Fatalf("NextState(%s, %v) = %s, want %s", tc.start, tc.tms, got, tc.end)
		}
	}
}

func TestStateMachineReset(t *testing.T) {
	m := NewStateMachine()
	// Move out of reset to ensure Reset() actually travels back.
	m.Clock(false) // -> Run-Test/Idle
	if m.State() != StateRunTestIdle {
		t.Fatalf("State() = %s, want %s", m.State(), StateRunTestIdle)
	}

	seq := m.Reset()

	if len(seq.TMS) != 5 {
		t.Fatalf("Reset sequence length = %d, want 5", len(seq.TMS))
	}
	if want := StateTestLogicReset; m.State() != want {
		t.Fatalf("State after reset = %s, want %s", m.State(), want)
	}
	if seq.States[len(seq.States)-1] != StateTestLogicReset {
		t.Fatalf("Final sequence state = %s, want %s", seq.States[len(seq.States)-1], StateTestLogicReset)
	}
}

func TestGoToProducesExpectedPattern(t *testing.T) {
	m := NewStateMachine()
	// Move into Run-Test/Idle so GoTo has to traverse more than one edge.
	m.Clock(false)

	path, err := m.GoTo(StateShiftIR)
	if err != nil {
		t.Fatalf("GoTo returned error: %v", err)
	}

	wantBits := []bool{true, true, false, false}
	if len(path.TMS) != len(wantBits) {
		t.Fatalf("GoTo length = %d, want %d", len(path.TMS), len(wantBits))
	}
	for i, want := range wantBits {
		if path.TMS[i] != want {
			t.Fatalf("path bit %d = %v, want %v", i, path.TMS[i], want)
		}
	}
	if m.State() != StateShiftIR {
		t.Fatalf("State() = %s, want %s", m.State(), StateShiftIR)
	}

	// Go back to Run-Test/Idle to ensure BFS works from IR path.
	if _, err := m.GoTo(StateRunTestIdle); err != nil {
		t.Fatalf("GoTo RunTestIdle returned error: %v", err)
	}
	if m.State() != StateRunTestIdle {
		t.Fatalf("State() = %s, want %s", m.State(), StateRunTestIdle)
	}
}
