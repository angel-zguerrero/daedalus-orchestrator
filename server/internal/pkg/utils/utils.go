package utils

import (
	"time"
)

// HasDuplicates checks if a slice of strings contains any duplicate items.
// It uses a map to keep track of items seen so far.
//
// Parameters:
//   - items: A slice of strings to check for duplicates.
//
// Returns:
//   - true if duplicates are found, false otherwise.
func HasDuplicates(items []string) bool {
	seen := make(map[string]struct{})
	for _, item := range items {
		if _, ok := seen[item]; ok {
			return true
		}
		seen[item] = struct{}{}
	}
	return false
}

// Contains checks if a slice of strings Contains a specific target string.
//
// Parameters:
//   - list: The slice of strings to search within.
//   - target: The string to search for.
//
// Returns:
//   - true if the target string is found in the list, false otherwise.
func Contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
func GetNowInInt() int64 {
	now := time.Now()
	return now.UnixNano()
}
