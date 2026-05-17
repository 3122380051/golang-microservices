package observability

// Tracer is a placeholder for distributed tracing setup.
// OpenTelemetry can be wired here in later tasks.
type Tracer struct{}

// NewTracer returns a no-op tracer placeholder.
func NewTracer() *Tracer {
	return &Tracer{}
}
