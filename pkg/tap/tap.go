package tap

import (
	"fmt"
)

// State represents one of the 16 defined IEEE 1149.1 TAP controller states.
type State uint8

const (
	StateTestLogicReset State = iota
	StateRunTestIdle
	StateSelectDRScan
	StateCaptureDR
	StateShiftDR
	StateExit1DR
	StatePauseDR
	StateExit2DR
	StateUpdateDR
	StateSelectIRScan
	StateCaptureIR
	StateShiftIR
	StateExit1IR
	StatePauseIR
	StateExit2IR
	StateUpdateIR
)

var stateNames = map[State]string{
	StateTestLogicReset: "TestLogicReset",
	StateRunTestIdle:    "RunTestIdle",
	StateSelectDRScan:   "SelectDRScan",
	StateCaptureDR:      "CaptureDR",
	StateShiftDR:        "ShiftDR",
	StateExit1DR:        "Exit1DR",
	StatePauseDR:        "PauseDR",
	StateExit2DR:        "Exit2DR",
	StateUpdateDR:       "UpdateDR",
	StateSelectIRScan:   "SelectIRScan",
	StateCaptureIR:      "CaptureIR",
	StateShiftIR:        "ShiftIR",
	StateExit1IR:        "Exit1IR",
	StatePauseIR:        "PauseIR",
	StateExit2IR:        "Exit2IR",
	StateUpdateIR:       "UpdateIR",
}

func (s State) String() string {
	if name, ok := stateNames[s]; ok {
		return name
	}
	return fmt.Sprintf("State(%d)", s)
}

// Sequence captures the TMS drive pattern and the sequence of states that result
// from applying that pattern to the TAP controller.
type Sequence struct {
	TMS    []bool
	States []State
}

type stateTransitions struct {
	onZero State
	onOne  State
}

var transitions = map[State]stateTransitions{
	StateTestLogicReset: {onZero: StateRunTestIdle, onOne: StateTestLogicReset},
	StateRunTestIdle:    {onZero: StateRunTestIdle, onOne: StateSelectDRScan},
	StateSelectDRScan:   {onZero: StateCaptureDR, onOne: StateSelectIRScan},
	StateCaptureDR:      {onZero: StateShiftDR, onOne: StateExit1DR},
	StateShiftDR:        {onZero: StateShiftDR, onOne: StateExit1DR},
	StateExit1DR:        {onZero: StatePauseDR, onOne: StateUpdateDR},
	StatePauseDR:        {onZero: StatePauseDR, onOne: StateExit2DR},
	StateExit2DR:        {onZero: StateShiftDR, onOne: StateUpdateDR},
	StateUpdateDR:       {onZero: StateRunTestIdle, onOne: StateSelectDRScan},
	StateSelectIRScan:   {onZero: StateCaptureIR, onOne: StateTestLogicReset},
	StateCaptureIR:      {onZero: StateShiftIR, onOne: StateExit1IR},
	StateShiftIR:        {onZero: StateShiftIR, onOne: StateExit1IR},
	StateExit1IR:        {onZero: StatePauseIR, onOne: StateUpdateIR},
	StatePauseIR:        {onZero: StatePauseIR, onOne: StateExit2IR},
	StateExit2IR:        {onZero: StateShiftIR, onOne: StateUpdateIR},
	StateUpdateIR:       {onZero: StateRunTestIdle, onOne: StateSelectDRScan},
}

// NextState returns the next TAP state after clocking TCK with the provided TMS
// value. It panics if an invalid state is supplied, which should never happen
// when interacting through the exported API.
func NextState(current State, tms bool) State {
	row, ok := transitions[current]
	if !ok {
		panic(fmt.Sprintf("tap: unhandled state %d", current))
	}
	if tms {
		return row.onOne
	}
	return row.onZero
}

// StateMachine tracks the TAP controller state locally. It does not perform any
// I/O; instead it produces the sequences of TMS bits needed so a hardware
// adapter can be instructed separately.
type StateMachine struct {
	state State
}

// NewStateMachine creates a TAP state machine initialized to Test-Logic-Reset.
func NewStateMachine() *StateMachine {
	return &StateMachine{state: StateTestLogicReset}
}

// State reports the current TAP state tracked by the machine.
func (m *StateMachine) State() State {
	return m.state
}

// Clock advances the machine one TCK cycle with the provided TMS bit and
// returns the new state.
func (m *StateMachine) Clock(tms bool) State {
	next := NextState(m.state, tms)
	m.state = next
	return next
}

// Reset applies the IEEE recommendation of clocking five consecutive TMS=1
// cycles. It returns the sequence for convenience so it can be forwarded to a
// hardware adapter.
func (m *StateMachine) Reset() Sequence {
	seq := Sequence{
		TMS:    make([]bool, 5),
		States: make([]State, 6),
	}
	seq.States[0] = m.state
	for i := 0; i < 5; i++ {
		seq.TMS[i] = true
		seq.States[i+1] = m.Clock(true)
	}
	return seq
}

// GoTo computes the minimal sequence of TMS values needed to reach the target
// state from the current state. It updates the machine as a side effect and
// returns the generated sequence.
func (m *StateMachine) GoTo(target State) (Sequence, error) {
	path, err := computePath(m.state, target)
	if err != nil {
		return Sequence{}, err
	}
	for _, bit := range path.TMS {
		m.Clock(bit)
	}
	return path, nil
}

// computePath uses BFS across the TAP state diagram to find the shortest set of
// transitions between two states.
func computePath(from, to State) (Sequence, error) {
	if _, ok := transitions[from]; !ok {
		return Sequence{}, fmt.Errorf("tap: invalid start state %d", from)
	}
	if _, ok := transitions[to]; !ok {
		return Sequence{}, fmt.Errorf("tap: invalid target state %d", to)
	}
	if from == to {
		return Sequence{States: []State{from}}, nil
	}

	type node struct {
		state  State
		tms    []bool
		states []State
	}

	queue := []node{{
		state:  from,
		tms:    nil,
		states: []State{from},
	}}
	visited := map[State]struct{}{from: {}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		nextStates := []struct {
			bit  bool
			next State
		}{
			{bit: false, next: NextState(current.state, false)},
			{bit: true, next: NextState(current.state, true)},
		}

		for _, candidate := range nextStates {
			if _, seen := visited[candidate.next]; seen {
				continue
			}

			newTMS := append(append([]bool{}, current.tms...), candidate.bit)
			newStates := append(append([]State{}, current.states...), candidate.next)

			if candidate.next == to {
				return Sequence{
					TMS:    newTMS,
					States: newStates,
				}, nil
			}

			visited[candidate.next] = struct{}{}
			queue = append(queue, node{
				state:  candidate.next,
				tms:    newTMS,
				states: newStates,
			})
		}
	}

	return Sequence{}, fmt.Errorf("tap: no path from %s to %s", from, to)
}
