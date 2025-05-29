package utils

// IsValidPort checks if a given integer p represents a valid, non-privileged network port number.
// Valid ports are typically in the range 1024 to 65535.
// Ports below 1024 are considered privileged and usually require special permissions to use.
//
// Parameters:
//   - p: The port number as an integer.
//
// Returns:
//   - true if the port number is within the valid range (1024-65535).
//   - false otherwise.
func IsValidPort(p int) bool {
	// Ports are generally considered valid in the range 1-65535.
	// Ports 1-1023 are well-known/privileged ports.
	// This function checks for common user-space application ports.
	return p >= 1024 && p <= 65535
}
