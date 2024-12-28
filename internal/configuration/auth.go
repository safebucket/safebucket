package configuration

// TODO: "users", CHANGE IT
var excludedAuthPaths = [2]string{"/auth", "/users"}

func GetExcludedAuthPaths() [2]string {
	return excludedAuthPaths
}
