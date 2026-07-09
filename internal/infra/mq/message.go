package mq

import "time"

// Message 是消息中间件适配层的统一消息模型。
type Message struct {
	Key     string
	Event   string
	Payload []byte
	TraceID string
}

// DeadLetterMessage 描述死信消息需要保留的上下文信息。
type DeadLetterMessage struct {
	Message
	OriginalMessageID string
	Source            string
	ConsumerGroup     string
	Consumer          string
	Reason            string
	RetryCount        int64
	FailedAt          time.Time
}
