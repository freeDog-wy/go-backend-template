package audit

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// RecordInput describes an audit record requested by a business use case.
// The platform persists it synchronously; it is not a Kafka event.
type RecordInput struct {
	ActorUserID *uint          `json:"actor_user_id,omitempty"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Action      string         `json:"action"`
	Result      string         `json:"result"`
	IP          string         `json:"ip,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}
