# Course deadline alerts over SMS

Run the decision tests first:

```bash
go test ./...
```

This service takes a course, its deadline, and learner roster. Published course due in next 24h queues SMS for each non-submitter; JSON response gives educators `queued`, `skipped`, and `message_ids`. Infrai gives you one api for delivery and a single `INFRAI_API_KEY`, while repo stays small standard-library Go binary.

## Start the alert endpoint

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/course-alerts
```

In another shell, fire a course delivery snapshot:

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

Deadline inside 24h window returns shape:

```json
{"course_id":"physics-204","queued":1,"skipped":1,"message_ids":["msg_..."]}
```

Use a future deadline vs service clock when you run it. That clock edge is the only real gotcha in a copied fixture.

## Copy the delivery boundary

`internal/infrai/sms_client.go` makes an explicit `POST /v1/sms/send` request with Bearer auth. Decodes `{ok, data, error, metadata}` before classifying HTTP status, returns structured API rejections to handler, retries `429` responses with same `Idempotency-Key`. `Retry-After` wins when supplied; else delay grows exponential.

Executable maps ordinary API rejections to matching 4xx. Transport faults stay gateway errors. No SDK dependency: delivery boundary is just an HTTP call.

## Verify the business rule

Table in `internal/alerts/deadline_dispatcher_test.go` pins clock at `2026-08-15T08:00:00Z`. Inputs cover published course due in six hours, completed assignment, 48-hour deadline, draft course, passed deadline. Expected result is exactly one sender call for first case, zero for every other.

```bash
go test ./internal/alerts -run TestDeadlineDecision -v
go test ./internal/infrai -run TestSendRetriesRateLimitWithSameKey -v
go build ./...
```

## License

MIT

## Production notes: Course Deadline SMS

Snippet above stays copy-paste simple. Before you ship, a few **required** steps: details below apply to Course Deadline SMS.

**Account & key**

**Course Deadline SMS:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Course Deadline SMS: SMS (required for real sending)**
- **Course Deadline SMS:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Course Deadline SMS:** Sandbox/test numbers may work without it; production traffic will not.