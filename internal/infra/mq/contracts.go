package mq

import "context"

// EventHandler 统一处理 Message，避免上层直接依赖某个中间件的原生消息结构。
type EventHandler func(ctx context.Context, message Message) error

// Publisher 定义消息发布适配层的统一契约。
type Publisher interface {
	Publish(ctx context.Context, message Message) error
}

// Consumer 定义消息消费适配层的统一契约。
type Consumer interface {
	Handle(eventName string, fn EventHandler)
	Run(ctx context.Context) error
}
