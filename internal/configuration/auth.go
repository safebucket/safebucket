package configuration

var excludedAuthPaths = [1]string{"/auth"}

func GetExcludedAuthPaths() [1]string {
	return excludedAuthPaths
}
