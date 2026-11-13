# Reminder delivery semantics

Reminder delivery is **at least once**. A request atomically claims the tuple
`(user_id, invoice_id, template_id, reminder_type, idempotency_key)` with a
five-minute lease. Concurrent requests while the lease is live return the same
`sending` reminder and do not send. `sent`, `skipped`, and `bounced` records are
terminal and are returned unchanged. Failed attempts and expired leases can be
claimed again and increment `attempt_count`.

A process can fail after the provider accepted a message but before `MarkSent`
commits. Once its lease expires, a retry may therefore produce a duplicate;
this is the unavoidable ambiguity of an external provider without a provider
idempotency API. The lease prevents immediate/concurrent duplicate sends while
ensuring abandoned work and `MarkSent` failures remain recoverable.

The provider is an explicit per-user setting (`gmail`, `microsoft`, or `smtp`).
There is no fallback: missing credentials/configuration fails that attempt
instead of sending from another account.
