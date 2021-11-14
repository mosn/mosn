package rules

func init() {
	MustRegister(&simpleMatcher{})
}

// macther factory
var transferMatcher TransferMatcher

func MustRegister(matcher TransferMatcher) {
	transferMatcher = matcher
}

func GetMatcher() TransferMatcher {
	return transferMatcher
}
