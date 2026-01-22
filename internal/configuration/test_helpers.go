package configuration

// SetMFABypassRulesForTesting allows tests to modify MFABypassRules.
// This function should only be used in test code.
func SetMFABypassRulesForTesting(rules []MFABypassRule) {
	MFABypassRules = rules
}
