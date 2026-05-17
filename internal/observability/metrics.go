package observability

import "expvar"

var (
	RequestCount = expvar.NewInt("http_requests_total")
	RequestErrors = expvar.NewInt("http_request_errors_total")
)
