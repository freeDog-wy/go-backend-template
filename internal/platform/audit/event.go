package audit

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

type LogRequested struct {
	ActorUserID *uint          `json:"actor_user_id,omitempty"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Action      string         `json:"action"`
	Result      string         `json:"result"`
	IP          string         `json:"ip,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (LogRequested) EventName() string { return "audit.log.requested" }
