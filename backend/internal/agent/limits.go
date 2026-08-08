package agent

import "time"

const (
	MaxToolCallsPerTurn = 20
	MaxContextMessages  = 20
	MaxMessageTokens    = 4000
	ConfirmationTimeout = 30 * time.Minute
)
