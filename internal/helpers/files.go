package helpers

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
)

// TODO: Currently not used, need to retrieve last id in database
func IncrementFileName(filename string) string {
	ext := filepath.Ext(filename)
	name := filename[:len(filename)-len(ext)]

	re := regexp.MustCompile(`^(.*)\((\d+)\)$`)
	matches := re.FindStringSubmatch(name)

	if len(matches) > 0 {
		baseName := matches[1]
		currentNum, _ := strconv.Atoi(matches[2])
		newNum := currentNum + 1
		return fmt.Sprintf("%s(%d)%s", baseName, newNum, ext)
	} else {
		return fmt.Sprintf("%s(1)%s", name, ext)
	}
}
