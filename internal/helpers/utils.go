package helpers

import "strings"

func FilterStringsByPrefix(input []string, prefix string) []string {
	var filtered []string
	for _, str := range input {
		if strings.HasPrefix(str, prefix) {
			filtered = append(filtered, str)
		}
	}
	return filtered
}
