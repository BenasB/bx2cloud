package exits

type ExitCode int

const (
	SUCCESS ExitCode = iota
	MISSING_COMMAND
	SERVER_ERROR
	UNKNOWN_COMMAND
	GREET_ERROR
)
