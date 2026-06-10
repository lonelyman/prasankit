// Package deliverablehandler exposes HTTP routes for the deliverable module (M3 —
// งวดงาน + submissions + ตรวจรับ).
package deliverablehandler

import (
	"errors"
	"time"

	"prasankit-api/internal/modules/deliverable"
	"prasankit-api/internal/modules/workspace"
	"prasankit-api/internal/transport/http/presenter"
	workspacehandler "prasankit-api/internal/transport/http/workspace"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// Handler handles /workspaces/projects/:projectID/deliverables routes.
type Handler struct {
	svc *deliverable.Service
}

// NewHandler constructs a Handler.
func NewHandler(svc *deliverable.Service) *Handler { return &Handler{svc: svc} }

// ── Request types ────────────────────────────────────────────────────────────

type createDeliverableRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"due_date"` // "YYYY-MM-DD"
	SortOrder   *int    `json:"sort_order"`
}

type updateDeliverableRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	DueDate     *string `json:"due_date"`
	SortOrder   *int    `json:"sort_order"`
}

type submitRequest struct {
	Note *string `json:"note"`
	URL  *string `json:"url"`
	// NOTE: submitted_by intentionally absent — server-derived from the actor's project_member
	// (D60/§5.4). Any client-supplied value is ignored silently.
}

type reviewRequest struct {
	DecisionCode string  `json:"decision_code"`
	Comment      *string `json:"comment"`
	// NOTE: reviewed_by intentionally absent — server-derived (D60/§5.4).
}

// ── Response types ────────────────────────────────────────────────────────────

type reviewResponse struct {
	ID           string  `json:"id"`
	DecisionCode string  `json:"decision_code"`
	Comment      *string `json:"comment"`
	ReviewedBy   string  `json:"reviewed_by"`
	ReviewedAt   string  `json:"reviewed_at"`
}

type submissionResponse struct {
	ID             string          `json:"id"`
	RoundNo        int             `json:"round_no"`
	Note           *string         `json:"note"`
	URL            *string         `json:"url"`
	SubmittedBy    string          `json:"submitted_by"`
	SubmittedAt    string          `json:"submitted_at"`
	TimelinessCode string          `json:"timeliness_code"`
	LateByDays     int             `json:"late_by_days"`
	Review         *reviewResponse `json:"review"`
}

type deliverableResponse struct {
	ID                    string              `json:"id"`
	ProjectID             string              `json:"project_id"`
	Title                 string              `json:"title"`
	Description           *string             `json:"description"`
	DueDate               *string             `json:"due_date"`
	SortOrder             int                 `json:"sort_order"`
	DeliverableStatusCode string              `json:"deliverable_status_code"`
	LatestSubmission      *submissionResponse `json:"latest_submission"`
	CreatedAt             string              `json:"created_at"`
	UpdatedAt             string              `json:"updated_at"`
}

// deliverableDetailResponse adds the full submissions history (round DESC) to a deliverable.
type deliverableDetailResponse struct {
	deliverableResponse
	Submissions []submissionResponse `json:"submissions"`
}

type listDeliverablesResponse struct {
	Items []deliverableResponse `json:"items"`
}

// ── mappers (timeliness derived here at the presenter, D62) ──────────────────────

// toSubmissionResponse maps a submission+review, deriving its timeliness against dueDate.
func toSubmissionResponse(swr deliverable.SubmissionWithReview, dueDate *time.Time) submissionResponse {
	code, lateBy := deliverable.DeriveTimeliness(dueDate, swr.SubmittedAt)
	resp := submissionResponse{
		ID:             swr.ID.String(),
		RoundNo:        swr.RoundNo,
		Note:           swr.Note,
		URL:            swr.URL,
		SubmittedBy:    swr.SubmittedBy.String(),
		SubmittedAt:    swr.SubmittedAt.UTC().Format(time.RFC3339),
		TimelinessCode: code,
		LateByDays:     lateBy,
	}
	if swr.Review != nil {
		resp.Review = &reviewResponse{
			ID:           swr.Review.ID.String(),
			DecisionCode: swr.Review.DecisionCode,
			Comment:      swr.Review.Comment,
			ReviewedBy:   swr.Review.ReviewedBy.String(),
			ReviewedAt:   swr.Review.ReviewedAt.UTC().Format(time.RFC3339),
		}
	}
	return resp
}

func toDeliverableResponse(dws deliverable.DeliverableWithStatus) deliverableResponse {
	resp := deliverableResponse{
		ID:                    dws.ID.String(),
		ProjectID:             dws.ProjectID.String(),
		Title:                 dws.Title,
		Description:           dws.Description,
		SortOrder:             dws.SortOrder,
		DeliverableStatusCode: deliverable.DeriveStatus(dws.LatestSubmission),
		CreatedAt:             dws.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:             dws.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if dws.DueDate != nil {
		s := dws.DueDate.UTC().Format("2006-01-02")
		resp.DueDate = &s
	}
	if dws.LatestSubmission != nil {
		sub := toSubmissionResponse(*dws.LatestSubmission, dws.DueDate)
		resp.LatestSubmission = &sub
	}
	return resp
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func tenantOf(c fiber.Ctx) (*workspace.TenantContext, bool) {
	tc, ok := c.Locals(workspacehandler.LocalsKeyTenant).(*workspace.TenantContext)
	if !ok || tc == nil {
		return nil, false
	}
	return tc, true
}

// HandleCreate handles POST /api/v1/workspaces/projects/:projectID/deliverables.
func (h *Handler) HandleCreate(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, err := uuid.Parse(c.Params("projectID"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	var req createDeliverableRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}
	due, fe := parseDate("due_date", req.DueDate)
	if fe != nil {
		return renderFieldError(c, *fe)
	}

	d, err := h.svc.CreateDeliverable(c.Context(), deliverable.CreateDeliverableInput{
		TenantCtx:   *tc,
		ProjectID:   projectID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     due,
		SortOrder:   req.SortOrder,
		IP:          c.IP(),
		UserAgent:   string(c.Request().Header.UserAgent()),
		RequestID:   c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	// Freshly-created deliverable has no submissions yet → not_submitted.
	return presenter.RenderItem(c, toDeliverableResponse(deliverable.DeliverableWithStatus{Deliverable: *d}), fiber.StatusCreated)
}

// HandleList handles GET /api/v1/workspaces/projects/:projectID/deliverables.
func (h *Handler) HandleList(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, err := uuid.Parse(c.Params("projectID"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid project id")
	}

	items, err := h.svc.ListDeliverables(c.Context(), deliverable.ListDeliverablesInput{
		TenantCtx: *tc,
		ProjectID: projectID,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	respItems := make([]deliverableResponse, 0, len(items))
	for _, d := range items {
		respItems = append(respItems, toDeliverableResponse(d))
	}
	return presenter.RenderItem(c, listDeliverablesResponse{Items: respItems})
}

// HandleGet handles GET /api/v1/workspaces/projects/:projectID/deliverables/:deliverableID.
func (h *Handler) HandleGet(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, deliverableID, fe := parseIDs(c)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", *fe)
	}

	detail, err := h.svc.GetDeliverable(c.Context(), deliverable.GetDeliverableInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		DeliverableID: deliverableID,
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}

	base := toDeliverableResponse(detail.DeliverableWithStatus)
	subs := make([]submissionResponse, 0, len(detail.Submissions))
	for _, swr := range detail.Submissions {
		subs = append(subs, toSubmissionResponse(swr, detail.DueDate))
	}
	return presenter.RenderItem(c, deliverableDetailResponse{
		deliverableResponse: base,
		Submissions:         subs,
	})
}

// HandleUpdate handles PUT /api/v1/workspaces/projects/:projectID/deliverables/:deliverableID.
func (h *Handler) HandleUpdate(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, deliverableID, fe := parseIDs(c)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", *fe)
	}

	var req updateDeliverableRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}
	due, fde := parseDate("due_date", req.DueDate)
	if fde != nil {
		return renderFieldError(c, *fde)
	}

	d, err := h.svc.UpdateDeliverable(c.Context(), deliverable.UpdateDeliverableInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		DeliverableID: deliverableID,
		Title:         req.Title,
		Description:   req.Description,
		DueDate:       due,
		SortOrder:     req.SortOrder,
		IP:            c.IP(),
		UserAgent:     string(c.Request().Header.UserAgent()),
		RequestID:     c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, toDeliverableResponse(deliverable.DeliverableWithStatus{Deliverable: *d}))
}

// HandleDelete handles DELETE /api/v1/workspaces/projects/:projectID/deliverables/:deliverableID.
func (h *Handler) HandleDelete(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, deliverableID, fe := parseIDs(c)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", *fe)
	}

	err := h.svc.DeleteDeliverable(c.Context(), deliverable.DeleteDeliverableInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		DeliverableID: deliverableID,
		IP:            c.IP(),
		UserAgent:     string(c.Request().Header.UserAgent()),
		RequestID:     c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// HandleSubmit handles POST .../deliverables/:deliverableID/submissions.
func (h *Handler) HandleSubmit(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, deliverableID, fe := parseIDs(c)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", *fe)
	}

	var req submitRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	sub, err := h.svc.Submit(c.Context(), deliverable.SubmitInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		DeliverableID: deliverableID,
		Note:          req.Note,
		URL:           req.URL,
		IP:            c.IP(),
		UserAgent:     string(c.Request().Header.UserAgent()),
		RequestID:     c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	// Newly-submitted round has no review yet; timeliness needs the deliverable's due_date,
	// which the service does not return here. The client re-fetches the deliverable for the
	// full derived view; the 201 echoes the created submission's own fields (no timeliness/
	// review yet by construction — round just created, unreviewed).
	return presenter.RenderItem(c, submitToResponse(*sub), fiber.StatusCreated)
}

// HandleReview handles POST .../submissions/:submissionID/review.
func (h *Handler) HandleReview(c fiber.Ctx) error {
	tc, ok := tenantOf(c)
	if !ok {
		return presenter.RenderError(c, fiber.StatusUnauthorized, "auth.unauthenticated", "Not authenticated")
	}
	projectID, deliverableID, fe := parseIDs(c)
	if fe != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", *fe)
	}
	submissionID, err := uuid.Parse(c.Params("submissionID"))
	if err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid submission id")
	}

	var req reviewRequest
	if err := c.Bind().JSON(&req); err != nil {
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Invalid request body")
	}

	rv, err := h.svc.Review(c.Context(), deliverable.ReviewInput{
		TenantCtx:     *tc,
		ProjectID:     projectID,
		DeliverableID: deliverableID,
		SubmissionID:  submissionID,
		DecisionCode:  req.DecisionCode,
		Comment:       req.Comment,
		IP:            c.IP(),
		UserAgent:     string(c.Request().Header.UserAgent()),
		RequestID:     c.Get("X-Request-Id"),
	})
	if err != nil {
		return h.handleServiceError(c, err)
	}
	return presenter.RenderItem(c, reviewResponse{
		ID:           rv.ID.String(),
		DecisionCode: rv.DecisionCode,
		Comment:      rv.Comment,
		ReviewedBy:   rv.ReviewedBy.String(),
		ReviewedAt:   rv.ReviewedAt.UTC().Format(time.RFC3339),
	}, fiber.StatusCreated)
}

// submitToResponse maps a just-created submission (no review, timeliness not computed without
// due_date — the create echo omits it; the list/detail derive view carries the full picture).
func submitToResponse(s deliverable.Submission) submissionResponse {
	return submissionResponse{
		ID:          s.ID.String(),
		RoundNo:     s.RoundNo,
		Note:        s.Note,
		URL:         s.URL,
		SubmittedBy: s.SubmittedBy.String(),
		SubmittedAt: s.SubmittedAt.UTC().Format(time.RFC3339),
		// TimelinessCode left empty + LateByDays 0 + Review nil: a 201 submit echo cannot
		// derive timeliness (no due_date in the submission row); clients read the deliverable.
	}
}

// ── helpers ────────────────────────────────────────────────────────────────────

func parseIDs(c fiber.Ctx) (projectID, deliverableID uuid.UUID, errMsg *string) {
	pid, err := uuid.Parse(c.Params("projectID"))
	if err != nil {
		m := "Invalid project id"
		return uuid.Nil, uuid.Nil, &m
	}
	did, err := uuid.Parse(c.Params("deliverableID"))
	if err != nil {
		m := "Invalid deliverable id"
		return uuid.Nil, uuid.Nil, &m
	}
	return pid, did, nil
}

// parseDate parses a YYYY-MM-DD string into UTC-midnight time.Time (same invariant as project).
func parseDate(field string, s *string) (*time.Time, *deliverable.FieldError) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, &deliverable.FieldError{Field: field, Message: "must be a YYYY-MM-DD date"}
	}
	return &t, nil
}

func renderFieldError(c fiber.Ctx, fe deliverable.FieldError) error {
	return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
		map[string]any{"fields": []deliverable.FieldError{fe}})
}

// handleServiceError maps deliverable service errors to HTTP responses (stable codes).
func (h *Handler) handleServiceError(c fiber.Ctx, err error) error {
	var valErr *deliverable.ValidationError
	switch {
	case errors.As(err, &valErr):
		return presenter.RenderError(c, fiber.StatusBadRequest, "validation.invalid_input", "Validation failed",
			map[string]any{"fields": valErr.Fields})
	case errors.Is(err, deliverable.ErrProjectNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "project.not_found", "Project not found")
	case errors.Is(err, deliverable.ErrDeliverableNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "deliverable.not_found", "Deliverable not found")
	case errors.Is(err, deliverable.ErrSubmissionNotFound):
		return presenter.RenderError(c, fiber.StatusNotFound, "submission.not_found", "Submission not found")
	case errors.Is(err, deliverable.ErrForbidden):
		return presenter.RenderError(c, fiber.StatusForbidden, "deliverable.forbidden", "You do not have permission to perform this action")
	case errors.Is(err, deliverable.ErrNotProjectMember):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "deliverable.not_project_member",
			"You must be an active project member to submit or review (org admins must be added to the project first)")
	case errors.Is(err, deliverable.ErrAcceptTerminal):
		return presenter.RenderError(c, fiber.StatusConflict, "deliverable.accept_terminal",
			"This deliverable was already accepted and cannot accept further submissions")
	case errors.Is(err, deliverable.ErrSubmissionSuperseded):
		return presenter.RenderError(c, fiber.StatusConflict, "submission.superseded",
			"A newer submission round exists; only the latest round can be reviewed")
	case errors.Is(err, deliverable.ErrAlreadyReviewed):
		return presenter.RenderError(c, fiber.StatusConflict, "submission.already_reviewed",
			"This submission has already been reviewed")
	case errors.Is(err, deliverable.ErrInvalidMasterCode):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "deliverable.invalid_master_code",
			"Invalid decision_code", map[string]any{"field": "decision_code"})
	case errors.Is(err, deliverable.ErrSelfReviewForbidden):
		return presenter.RenderError(c, fiber.StatusUnprocessableEntity, "submission.self_review_forbidden",
			"You cannot review your own submission")
	default:
		return presenter.RenderError(c, fiber.StatusInternalServerError, "internal.unexpected", "Internal error")
	}
}
