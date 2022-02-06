package azuretts

import "github.com/joomcode/errorx"

var (
	Errors = errorx.NewNamespace("azuretts")

	TooManyRequests = errorx.NewType(Errors, "too_many_requests").ApplyModifiers(errorx.TypeModifierOmitStackTrace)
)
