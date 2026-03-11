package health

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusUnknown   Status = "unknown"
)

type CheckResult struct {
	Name    string
	Status  Status
	Message string
}

type ClusterHealth struct {
	ClusterName string
	Results     []CheckResult
}

type HealthSummary struct {
	Healthy   int
	Unhealthy int
	Unknown   int
	Total     int
}

func (ch *ClusterHealth) IsHealthy() bool {
	for _, r := range ch.Results {
		if r.Status != StatusHealthy {
			return false
		}
	}
	return true
}

func (ch *ClusterHealth) Summary() HealthSummary {
	s := HealthSummary{
		Total: len(ch.Results),
	}
	for _, r := range ch.Results {
		switch r.Status {
		case StatusHealthy:
			s.Healthy++
		case StatusUnhealthy:
			s.Unhealthy++
		case StatusUnknown:
			s.Unknown++
		}
	}
	return s
}
