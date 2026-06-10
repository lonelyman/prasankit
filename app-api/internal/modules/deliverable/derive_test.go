package deliverable_test

import (
	"testing"
	"time"

	"prasankit-api/internal/modules/deliverable"

	"github.com/google/uuid"
)

func submissionWithReview(decision string) *deliverable.SubmissionWithReview {
	swr := &deliverable.SubmissionWithReview{
		Submission: deliverable.Submission{ID: uuid.New(), RoundNo: 1},
	}
	if decision != "" {
		swr.Review = &deliverable.SubmissionReview{ID: uuid.New(), DecisionCode: decision}
	}
	return swr
}

// ── DeriveStatus: every branch ───────────────────────────────────────────────────

func TestDeriveStatus_AllBranches(t *testing.T) {
	tests := []struct {
		name  string
		input *deliverable.SubmissionWithReview
		want  string
	}{
		{"no submission → not_submitted", nil, deliverable.DeliverableStatusNotSubmitted},
		{"submission, no review → in_review", submissionWithReview(""), deliverable.DeliverableStatusInReview},
		{"review accepted → accepted", submissionWithReview(deliverable.SubmissionDecisionAccepted), deliverable.DeliverableStatusAccepted},
		{"review conditional → conditional", submissionWithReview(deliverable.SubmissionDecisionConditional), deliverable.DeliverableStatusConditional},
		{"review rejected → rejected", submissionWithReview(deliverable.SubmissionDecisionRejected), deliverable.DeliverableStatusRejected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliverable.DeriveStatus(tt.input); got != tt.want {
				t.Errorf("DeriveStatus = %q, want %q", got, tt.want)
			}
		})
	}
}

// An unexpected (non-master) decision code must pass through verbatim (fail-loud), not be
// coerced to in_review — guards against masking a master drift.
func TestDeriveStatus_UnknownDecisionPassesThrough(t *testing.T) {
	got := deliverable.DeriveStatus(submissionWithReview("mystery_code"))
	if got != "mystery_code" {
		t.Errorf("DeriveStatus(unknown) = %q, want passthrough %q", got, "mystery_code")
	}
}

// ── DeriveTimeliness: early / on_time / late + late_by_days / no_due + UTC boundary ─

func d(y int, m time.Month, day int) time.Time {
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
}

func TestDeriveTimeliness_NoDue(t *testing.T) {
	code, late := deliverable.DeriveTimeliness(nil, d(2026, 6, 10))
	if code != deliverable.TimelinessNoDue || late != 0 {
		t.Errorf("no_due: got (%q, %d), want (%q, 0)", code, late, deliverable.TimelinessNoDue)
	}
}

func TestDeriveTimeliness_Early(t *testing.T) {
	due := d(2026, 6, 10)
	code, late := deliverable.DeriveTimeliness(&due, d(2026, 6, 8))
	if code != deliverable.TimelinessEarly || late != 0 {
		t.Errorf("early: got (%q, %d), want (%q, 0)", code, late, deliverable.TimelinessEarly)
	}
}

func TestDeriveTimeliness_OnTime(t *testing.T) {
	due := d(2026, 6, 10)
	code, late := deliverable.DeriveTimeliness(&due, d(2026, 6, 10))
	if code != deliverable.TimelinessOnTime || late != 0 {
		t.Errorf("on_time: got (%q, %d), want (%q, 0)", code, late, deliverable.TimelinessOnTime)
	}
}

func TestDeriveTimeliness_LateWithDays(t *testing.T) {
	due := d(2026, 6, 10)
	code, late := deliverable.DeriveTimeliness(&due, d(2026, 6, 13))
	if code != deliverable.TimelinessLate || late != 3 {
		t.Errorf("late: got (%q, %d), want (%q, 3)", code, late, deliverable.TimelinessLate)
	}
}

// UTC date boundary: a submission at 23:59:59 UTC on the due date is on_time (same calendar
// day), and at 00:00:00 UTC the next day is late by 1.
func TestDeriveTimeliness_UTCDateBoundary(t *testing.T) {
	due := d(2026, 6, 10)

	lateNight := time.Date(2026, 6, 10, 23, 59, 59, 0, time.UTC)
	if code, late := deliverable.DeriveTimeliness(&due, lateNight); code != deliverable.TimelinessOnTime || late != 0 {
		t.Errorf("23:59 same-day: got (%q, %d), want (on_time, 0)", code, late)
	}

	nextMidnight := time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	if code, late := deliverable.DeriveTimeliness(&due, nextMidnight); code != deliverable.TimelinessLate || late != 1 {
		t.Errorf("next 00:00: got (%q, %d), want (late, 1)", code, late)
	}

	// A submitted_at in a non-UTC zone is compared by its UTC calendar day. 2026-06-11 06:00
	// in UTC+8 is 2026-06-10 22:00 UTC → still on_time, not late.
	zoneEast := time.FixedZone("UTC+8", 8*3600)
	earlyMorningEast := time.Date(2026, 6, 11, 6, 0, 0, 0, zoneEast) // = 2026-06-10 22:00 UTC
	if code, late := deliverable.DeriveTimeliness(&due, earlyMorningEast); code != deliverable.TimelinessOnTime || late != 0 {
		t.Errorf("UTC+8 06:00 next day: got (%q, %d), want (on_time, 0) — must compare by UTC day", code, late)
	}
}

// ── cross-master string-equal (drift guard) ──────────────────────────────────────

// The three terminal-verdict codes are seeded in BOTH submission_decisions (000018) and
// deliverable_statuses (000019), and DeriveStatus maps decision→status by EXACT string. If a
// rename drifts one side, this test fails before the runtime derive silently mis-maps.
func TestCrossMaster_TerminalCodesStringEqual(t *testing.T) {
	if deliverable.SubmissionDecisionAccepted != deliverable.DeliverableStatusAccepted {
		t.Errorf("accepted drift: decision %q != status %q", deliverable.SubmissionDecisionAccepted, deliverable.DeliverableStatusAccepted)
	}
	if deliverable.SubmissionDecisionConditional != deliverable.DeliverableStatusConditional {
		t.Errorf("conditional drift: decision %q != status %q", deliverable.SubmissionDecisionConditional, deliverable.DeliverableStatusConditional)
	}
	if deliverable.SubmissionDecisionRejected != deliverable.DeliverableStatusRejected {
		t.Errorf("rejected drift: decision %q != status %q", deliverable.SubmissionDecisionRejected, deliverable.DeliverableStatusRejected)
	}
	// And the literal values must remain the canonical strings (pinning the contract).
	if deliverable.SubmissionDecisionAccepted != "accepted" ||
		deliverable.SubmissionDecisionConditional != "conditional" ||
		deliverable.SubmissionDecisionRejected != "rejected" {
		t.Error("decision code literals drifted from canonical accepted/conditional/rejected")
	}
}
