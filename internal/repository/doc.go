// Package repository 提供 PostgreSQL 持久化实现和共享事务传播机制。
//
// 子包承载各上下文的查询；根包只维护 GORM 事务上下文，以保证多个 Repository 的写入原子性。
package repository
