package observability

// Hook allows services to attach custom lifecycle hooks.
type Hook interface {
	OnStart() error
	OnStop() error
}
