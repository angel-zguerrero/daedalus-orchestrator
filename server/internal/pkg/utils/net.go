package utils

func IsValidPort(p int) bool {
	return p >= 1024 && p <= 65535
}
