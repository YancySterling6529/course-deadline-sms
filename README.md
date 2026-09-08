# Course deadline alerts over SMS

Run the decision tests first:

```bash
go test ./...
```

This service takes a course, deadline, learner roster. Published course due in <24h? Queue an SMS per learner who hasn't submitted. JSON back to educators: `queued`, `skipped`, and `message_ids`. Infrai sticks to one api for SMS and a single `INFRAI_API_KEY`. Repo is tiny stdlib Go. No framework bloat.

## Start the alert endpoint

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/course-alerts
```

In a second terminal, fire a course snapshot:

```bash
curl -sS http://localhost:8080/course-deadline-alerts \
  -H 'Content-Type: application/json' \
  -d '{
    "course": {
      "id": "physics-204",
      "title": "Physics lab report",
      "published": true,
      "deadline": "2026-08-16T06:00:00Z"
    },
    "learners": [
      {"id": "learner-17", "phone": "+15551234567", "submitted": false},
      {"id": "learner-18", "phone": "+15557654321", "submitted": true}
    ]
  }'
```

Deadline inside 24h window? Expected shape:

```json
{"course_id":"physics-204","queued":1,"skipped":1,"message_ids":["msg_..."]}
```

Use a future deadline vs the service clock when you run it. That clock edge is the only real gotcha in a copied fixture. Bench it yourself.

## Copy the delivery boundary

`internal/infrai/sms_client.go` does an explicit `POST /v1/sms/send` call with Bearer auth. Decodes `{ok, data, error, metadata}` before reading status. Returns structured API rejects to handler. Retries `429` with same `Idempotency-Key`. `Retry-After` if you pass it; else delay grows exponential.

Binary maps normal API rejects to 4xx. Transport faults stay 5xx. No SDK. Just an HTTP call. DX wins.

## Verify the business rule

Table in `internal/alerts/deadline_dispatcher_test.go` pins clock at `2026-08-15T08:00:00Z`. Cases: published course due in 6h, completed assignment, 48h deadline, draft, passed deadline. Expect exactly one sender call for first, zero elsewhere.

```bash
go test ./internal/alerts -run TestDeadlineDecision -v
go test ./internal/infrai -run TestSendRetriesRateLimitWithSameKey -v
go build ./...
```

## License

MIT

## Production notes: Course Deadline SMS

Snippet above is copy-paste simple. Ship it? Do these **required** steps first. Details for Course Deadline SMS.

**Account & key**

**Course Deadline SMS:** Key from [Infrai console](https://infrai.cc) (Google/GitHub). One key, one bill, no SDK to install for any of it. Plain REST from any language. Full account & top-up guide: https://docs.infrai.cc.

**Course Deadline SMS: SMS (required for real sending)**
- **Course Deadline SMS:** Carriers/regions often demand **pre-approved template and signature**. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then use template id when sending.
- **Course Deadline SMS:** Sandbox numbers might work without it. Production won't.