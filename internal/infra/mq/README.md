# MQ Infra 语义说明

本文档说明 `internal/infra/mq` 当前的职责边界，以及 Redis / Kafka 两套实现的现状。

## 1. 目录职责

`internal/infra/mq` 只负责消息中间件适配，不负责业务编排。

当前目录分为两层：

- 抽象层
  - `message.go`
  - `contracts.go`
  - `outbox_adapter.go`
  - `errors.go`
- 中间件实现层
  - `publisher.go`
  - `kafka_publisher.go`
  - `consumer.go`
  - `kafka_consumer.go`
  - `kafka_support.go`

## 2. 统一契约

### 2.1 Message

统一消息模型：

```go
type Message struct {
    Key     string
    Event   string
    Payload []byte
    TraceID string
}
```

语义如下：

- `Key`
  - 稳定消息键。
  - 当前 outbox 链路默认使用 outbox event ID。
- `Event`
  - 事件名。
- `Payload`
  - 序列化后的业务载荷。
- `TraceID`
  - 链路追踪标识。

### 2.2 Publisher

统一发布契约：

```go
type Publisher interface {
    Publish(ctx context.Context, message Message) error
}
```

### 2.3 Consumer

统一消费契约：

```go
type Consumer interface {
    Handle(eventName string, fn EventHandler)
    Run(ctx context.Context) error
}
```

`EventHandler` 只处理统一消息模型：

```go
type EventHandler func(ctx context.Context, message Message) error
```

### 2.4 Outbox 适配器

`outbox_adapter.go` 把 `domain/outbox.Publisher` 适配成 `mq.Publisher`。

这样：

- `usecase/support/outbox_publisher.go` 继续依赖领域契约
- `cmd/cron` 只在装配时选择 RedisPublisher 或 KafkaPublisher

### 2.5 不可重试错误

`errors.go` 提供：

- `mq.MarkNonRetryable(err)`
- `mq.IsNonRetryable(err)`

Kafka consumer 会据此判断某个错误是否应直接进入 DLQ，而不是进入 retry topic。

## 3. 当前主链路

当前事件链路仍然是：

1. 业务事务内先写 `outbox_events`
2. `cmd/cron` 定时扫描 outbox
3. `cmd/cron` 通过 `mq.Publisher` 把消息投递到外部中间件
4. `cmd/worker` 通过 `mq.Consumer` 消费消息

职责划分如下：

- `outbox`
  - 负责本地事务一致性
- `mq`
  - 负责外部消息系统适配

## 4. 当前实现状态

### 4.1 RedisPublisher

`publisher.go` 把 `mq.Message` 映射成 Redis Streams 字段：

- `event`
- `data`
- `message_key`
- `trace_id`

### 4.2 KafkaPublisher

`kafka_publisher.go` 把 `mq.Message` 映射成 Kafka message：

- `Key` -> Kafka message key
- `Payload` -> Kafka message value
- `event` -> Kafka header
- `trace_id` -> Kafka header

### 4.3 RedisConsumer

`consumer.go` 当前除了读取 Redis Streams，还承担了 Redis 版可靠消费语义：

- `XREADGROUP` 读取新消息
- `XAUTOCLAIM` reclaim pending
- handler 失败不 `ack`
- 超过最大重试后写入 Redis DLQ stream
- 基于 Redis key 的消费幂等

### 4.4 KafkaConsumer

`kafka_consumer.go` 当前已经接入数据库消费记录表 `message_consumptions`，并补上了 Kafka 版基础失败流转：

- 先用 `consumer_group + message_key` 到 DB 抢占消费权
- 消费记录状态包括：
  - `processing`
  - `done`
  - `failed`
  - `dead`
- 如果记录已经是 `done` 或 `dead`
  - 直接提交当前 offset
- 如果记录仍在 `processing` 且锁未过期
  - 当前 worker 返回错误退出
- handler 成功后
  - 标记 `done`
  - 再提交 offset
- handler 失败后
  - 默认视为可重试错误
  - 未超过 `MaxRetries` 时转发到 retry topic
  - 超过 `MaxRetries` 时转发到 DLQ topic
- 如果错误被 `mq.MarkNonRetryable` 包装
  - 直接进入 DLQ topic
- malformed 消息
  - 不进入正常处理链路
  - 直接写入 DLQ topic 并提交 offset

当前 Kafka 会消费两个 topic：

- 主 topic
- retry topic

并会把死信写入：

- DLQ topic

## 5. DB 消费幂等解决了什么

Kafka 这一版已经不再依赖 Redis key 做幂等，而是依赖数据库消费记录表。

这带来了几个直接收益：

- 消费状态可审计
- 重启后仍然能判断某条消息是否已经处理
- 能明确区分 `done / failed / processing / dead`
- 为后续重试策略、DLQ 巡检和回放保留持久化基础

当前处理锁也从“Redis 临时锁”变成了“DB 记录上的 `locked_until`”。

## 6. Retry / DLQ 现状

Kafka 当前已经有：

- retry topic
- DLQ topic
- 最大重试次数控制
- 不可重试错误直达 DLQ

但这仍然是一版最小可用实现。

当前限制：

- retry topic 没有延迟退避
  - 失败消息会较快再次被消费
- 还没有分层 retry topic
  - 例如 `retry.1m / retry.10m / retry.1h`
- 还没有 DLQ 回放工具和巡检任务
- 还没有更细粒度的消费指标和报警

换句话说，当前实现已经有“失败流转”，但还没有“成熟的失败治理”。

## 7. 现阶段结论

现在这层已经具备：

- 统一消息契约
- Redis / Kafka 双发布实现
- Redis / Kafka 双消费实现
- Kafka 版 DB 消费幂等
- Kafka 版 retry topic / DLQ topic

但也要明确：

- Redis 版和 Kafka 版的可靠性语义仍然不统一
- Kafka 下一步更合适的方向是补“延迟重试”和“DLQ 运维闭环”
