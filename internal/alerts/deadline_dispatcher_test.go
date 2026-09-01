package alerts

import (
	"context"
	"testing"
	"time"

	"github.com/infrai-examples/course-deadline-sms/internal/infrai"
)

type recordingSender struct{ calls int }

func (s *recordingSender) Send(context.Context, string, string, string) (infrai.SendResult, error) {
	s.calls++
	return infrai.SendResult{MessageID: "msg-test"}, nil
}

func TestDeadlineDecision(t *testing.T) {
	now := time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		published bool
		due       time.Duration
		submitted bool
		wantCalls int
	}{
		{name: "published course due within 24 hours", published: true, due: 6 * time.Hour, wantCalls: 1},
		{name: "completed assignment", published: true, due: 6 * time.Hour, submitted: true, wantCalls: 0},
		{name: "deadline outside window", published: true, due: 48 * time.Hour, wantCalls: 0},
		{name: "draft course", published: false, due: 6 * time.Hour, wantCalls: 0},
		{name: "deadline passed", published: true, due: -time.Hour, wantCalls: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &recordingSender{}
			dispatcher := Dispatcher{Sender: sender, Now: func() time.Time { return now }}
			report, err := dispatcher.Dispatch(context.Background(), Request{
				Course:   Course{ID: "go-101", Title: "Go 101", Published: tt.published, Deadline: now.Add(tt.due)},
				Learners: []Learner{{ID: "learner-7", Phone: "+15551234567", Submitted: tt.submitted}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if sender.calls != tt.wantCalls || report.Queued != tt.wantCalls {
				t.Fatalf("calls=%d queued=%d, want %d", sender.calls, report.Queued, tt.wantCalls)
			}
		})
	}
}
