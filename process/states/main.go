package states

// Enumeração de estados possíveis de um processo
type State int

const (
	FREE State = 0
	WAIT State = 1
	HELD State = 2
)
