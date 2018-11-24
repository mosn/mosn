package router

type directResponseImpl struct {
	status int
	body   string
}

func (rule *directResponseImpl) StatusCode() int {
	return rule.status
}

func (rule *directResponseImpl) Body() string {
	return rule.body
}
