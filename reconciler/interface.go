package reconciler

// Reconciler defines interface used to reconcile
type Reconciler interface {
	// Reconcile reconciles local state with remote
	Reconcile() error
}
