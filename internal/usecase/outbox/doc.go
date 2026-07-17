// Package outbox 编排本地 Outbox 事件的发布任务。
//
// 本包由 cron 进程调用，扫描已提交但尚未确认发布的事件，并通过领域端口投递到外部
// 消息系统。它提供至少一次投递语义；业务 Usecase 只负责通过 shared.EventBus 写入
// Outbox，不应调用本包。
package outbox
