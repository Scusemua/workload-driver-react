package workload

type WorkloadStats struct {
	CumulativeNumStaticTrainingReplicas int `json:"CumulativeNumStaticTrainingReplicas"`
	TotalNumSessions                    int `json:"TotalNumSessions"`
}

func NewWorkloadStats() *WorkloadStats {
	return &WorkloadStats{}
}
