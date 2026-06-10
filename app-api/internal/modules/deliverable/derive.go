package deliverable

import "time"

// DeriveStatus computes the derived deliverable status code (D59) from the latest
// submission (with its review). It is a PURE function — the single source of truth for
// "what state is this งวด in" — never stored as a column.
//
//	latest == nil                      → not_submitted
//	latest present, no review          → in_review
//	latest present, review = accepted    → accepted
//	latest present, review = conditional → conditional
//	latest present, review = rejected    → rejected
//
// The review's decision_code is passed through verbatim (it is FK-validated against
// submission_decisions on write), so an unexpected code surfaces as-is rather than being
// silently coerced — fail-loud over wrong-fix.
func DeriveStatus(latest *SubmissionWithReview) string {
	if latest == nil {
		return DeliverableStatusNotSubmitted
	}
	if latest.Review == nil {
		return DeliverableStatusInReview
	}
	switch latest.Review.DecisionCode {
	case SubmissionDecisionAccepted:
		return DeliverableStatusAccepted
	case SubmissionDecisionConditional:
		return DeliverableStatusConditional
	case SubmissionDecisionRejected:
		return DeliverableStatusRejected
	default:
		// Should be unreachable: decision_code is FK-validated. Pass through so a master
		// drift is visible in the API surface instead of being masked as in_review.
		return latest.Review.DecisionCode
	}
}

// DeriveTimeliness computes the timeliness code (D62) by comparing the UTC calendar date
// of submitted_at against due_date. PURE function. (+late_by_days, days strictly positive
// only when late; 0 otherwise.)
//
//	due == nil                         → (no_due, 0)
//	date(submitted) <  due             → (early, 0)
//	date(submitted) == due             → (on_time, 0)
//	date(submitted) >  due             → (late, days_after)
//
// Comparison is by UTC calendar day (M3 OQ-3), NOT instant: both sides are truncated to
// midnight UTC so a 23:59 submission on the due date is on_time, not late. lateByDays counts
// whole UTC days the submission landed after the due date.
func DeriveTimeliness(dueDate *time.Time, submittedAt time.Time) (code string, lateByDays int) {
	if dueDate == nil {
		return TimelinessNoDue, 0
	}
	due := utcDay(*dueDate)
	sub := utcDay(submittedAt)
	switch {
	case sub.Before(due):
		return TimelinessEarly, 0
	case sub.Equal(due):
		return TimelinessOnTime, 0
	default:
		days := int(sub.Sub(due).Hours() / 24)
		return TimelinessLate, days
	}
}

// utcDay truncates t to midnight UTC (calendar-day granularity).
func utcDay(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}
