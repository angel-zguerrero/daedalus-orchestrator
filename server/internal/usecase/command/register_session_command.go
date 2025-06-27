package command

import (
	"deadalus-orch/server/internal/infrastructure/db"
	"encoding/gob"
	"time"
	// "fmt" // Will be needed if we add logging or more complex error handling
)

func init() {
	gob.Register(RegisterSessionCommand{})
}

// RegisterSessionCommand represents a command to register a user session
// using a JWT token.
type RegisterSessionCommand struct {
	JWTToken string
	// JWTKey is needed by SessionRepository to validate/parse the token.
	// Storing it in the command means it would be part of the Raft log.
	// This might have security implications and alternatives should be considered,
	// e.g., making it available to the Execute method via a context or environment.
	// For now, following the pattern of fields in LoginCommand.
	JWTKey []byte
}

// Execute registers the session using the SessionRepository.
// It requires an IDGeneratorFactory for the SessionRepository to create new session IDs.
// A DefaultIDGeneratorFactory is used here.
func (cmd *RegisterSessionCommand) Execute(uow *db.UnitOfWork, now time.Time) ([]byte, error) {
	if cmd.JWTToken == "" {
		return nil, NewDomainError("JWTToken cannot be empty for RegisterSessionCommand")
	}
	if len(cmd.JWTKey) == 0 {
		return nil, NewDomainError("JWTKey cannot be empty for RegisterSessionCommand")
	}

	// SessionRepository requires an IDGeneratorFactory.
	// Using DefaultIDGeneratorFactory as sessions will likely need unique IDs.
	idFactory := &db.DefaultIDGeneratorFactory{}

	sessionRepo, err := db.NewSessionRepository(uow, idFactory, cmd.JWTKey)
	if err != nil {
		return nil, NewInfrastructureErrorWrap(err, "failed to initialize SessionRepository")
	}

	err = sessionRepo.RegisterSession(cmd.JWTToken, now)
	if err != nil {
		// It could be a validation error (e.g. bad token) or a DB error.
		// Consider if different error types (DomainError vs InfrastructureError) are needed.
		// For now, wrapping it as a potential infrastructure or downstream error.
		return nil, NewDomainErrorWrap(err, "failed to register session")
	}

	// Command executed successfully, no specific data to return in the byte slice.
	return nil, nil
}

// --- Error Helper Types (assuming these or similar exist, or should be created) ---
// Based on common patterns, adding simple error types.
// If the project has a standard way to define domain/infrastructure errors, that should be used.

type DomainError struct {
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

func NewDomainError(message string) *DomainError {
	return &DomainError{Message: message}
}

func NewDomainErrorWrap(err error, message string) *DomainError {
	return &DomainError{Message: message, Cause: err}
}

type InfrastructureError struct {
	Message string
	Cause   error
}

func (e *InfrastructureError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// func NewInfrastructureError(message string) *InfrastructureError {
// 	return &InfrastructureError{Message: message}
// }

func NewInfrastructureErrorWrap(err error, message string) *InfrastructureError {
	return &InfrastructureError{Message: message, Cause: err}
}
