package plugin

type Output struct {
	Type  string
	Value interface{}
}

type Result struct {
	// Outputs is a list of outputs that the provisioner emitted.
	// This is available only when the provisioner has already run.
	// If the provisioner ended up just sending a repository_dispatch event to delegate the run to another prenv execution,
	// this field is nil.
	Outputs map[string]Output
}
