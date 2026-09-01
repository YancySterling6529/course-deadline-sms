package alerts

import (
	"context"
	"fmt"
	"time"

	"github.com/infrai-examples/course-deadline-sms/internal/infrai"
)

type Course struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Published bool      `json:"published"`
	Deadline  time.Time `json:"deadline"`
}

type Learner struct {
	ID        string `json:"id"`
	Phone     string `json:"phone"`
	Submitted bool   `json:"submitted"`
}

type Request struct {
	Course   Course    `json:"course"`
	Learners []Learner `json:"learners"`
}

type Report struct {
	CourseID string   `json:"course_id"`
	Queued   int      `json:"queued"`
	Skipped  int      `json:"skipped"`
	Messages []string `json:"message_ids"`
}

type Sender interface {
	Send(context.Context, string, string, string) (infrai.SendResult, error)
}

type Dispatcher struct {
	Sender Sender
	Now    func() time.Time
}

func (d Dispatcher) Dispatch(ctx context.Context, req Request) (Report, error) {
	report := Report{CourseID: req.Course.ID}
	now := d.Now().UTC()
	dueIn := req.Course.Deadline.UTC().Sub(now)
	eligibleCourse := req.Course.Published && dueIn > 0 && dueIn <= 24*time.Hour

	for _, learner := range req.Learners {
		if !eligibleCourse || learner.Submitted {
			report.Skipped++
			continue
		}
		message := fmt.Sprintf("%s is due %s UTC. Submit before the deadline.", req.Course.Title, req.Course.Deadline.UTC().Format("2006-01-02 15:04"))
		key := "course-deadline:" + req.Course.ID + ":" + learner.ID
		result, err := d.Sender.Send(ctx, learner.Phone, message, key)
		if err != nil {
			return report, err
		}
		report.Queued++
		report.Messages = append(report.Messages, result.MessageID)
	}
	return report, nil
}
