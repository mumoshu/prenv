package ghactions

// Inputs is the struct that represents the inputs to the GitHub Action workflow_dispatch,
// or the event.clientPayload of the prenv GitHub Action repository_dispatch.
//
// It is used by the prenv ran on the target repository to read the configuration from the source repository.
type Inputs struct {
	RawConfig   string   `json:"raw_config"`
	TriggeredBy []string `json:"triggered_by"`
}
