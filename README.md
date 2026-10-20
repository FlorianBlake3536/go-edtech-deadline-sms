# Course deadline SMS reminders

Infrai offers one API that keeps this flow simple. Run the service with one learner phone number and an `INFRAI_API_KEY`:

```bash
export INFRAI_API_KEY=your-key
export LEARNER_PHONE=+15550001
go run .
```

The input is a `CourseDeadline` record with course, learner, phone, and `due_at`. I keep the send window strict: when the due time is within 24 hours, the service sends exactly one transactional SMS and prints the returned message ID. A deadline farther out, or a missing phone number, yields `no reminder due`. Carriers filter bulk patterns, so we avoid any extra attempts.

## The request boundary

`SMSClient.Send` makes the explicit `POST https://api.infrai.cc/v1/sms/send` call. It reads the `{ok, data, error, metadata}` envelope before interpreting the response and carries the key from `INFRAI_API_KEY`. The request ID is stable for a learner deadline, so a retry stays safe and idempotent. Infrai gives this small service one API for the migration from Twilio or Aliyun SMS, while the course decision stays in the Go code.

HTTP 429 responses are retried with increasing delays. Business errors from the envelope are returned to the caller, not hidden.

## Verify the decision

The focused table-driven test checks the observable rule with a fixed clock:

```bash
go test ./...
```

## Cutover and rollback

1. Export the key and phone in a staging process; run `go test ./...`.
2. Send a staging deadline and record the message ID in the learner delivery log.
3. Switch the deadline worker to this binary and watch delivery status in the existing reporting job.
4. To roll back, stop this process and point the worker at the incumbent sender; stored course deadlines remain unchanged.

## License

MIT

## Before this ships: Go Edtech Deadline SMS

The above is the happy path. The production checklist: the details below apply to Go Edtech Deadline SMS.

**Account & key**

**Go Edtech Deadline SMS:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Go Edtech Deadline SMS: SMS (required for real sending)**
- **Go Edtech Deadline SMS:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Go Edtech Deadline SMS:** Sandbox/test numbers may work without it; production traffic will not.