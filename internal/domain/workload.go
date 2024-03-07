package domain

type WorkloadPreset struct {
	Name        string   `json:"name"`        // Human-readable name for this particular workload preset.
	Description string   `json:"description"` // Human-readable description of the workload.
	Key         string   `json:"key"`         // Key for code-use only (i.e., we don't intend to display this to the user for the most part).
	Months      []string `json:"months"`      // The months of data used by the workload.
}
