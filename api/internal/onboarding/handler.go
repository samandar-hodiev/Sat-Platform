package onboarding

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mounts /me/onboarding*. authenticate is the auth module's middleware.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	g := v1.Group("/me/onboarding", authenticate, learnerOnly)
	g.GET("", h.get)
	g.PUT("/goal", h.saveGoal)
	g.PUT("/schedule", h.saveSchedule)
	g.PUT("/background", h.saveBackground)
	g.POST("/complete", h.complete)
}

// learnerOnly: onboarding is a learner flow; the Owner has none.
func learnerOnly(c *gin.Context) {
	if auth.CurrentUser(c).Role != auth.RoleLearner {
		httpx.Fail(c, apperr.Forbidden("This is available to learners only."))
		return
	}
	c.Next()
}

type goalResponse struct {
	TargetScore   int       `json:"target_score"`
	ExamDate      *string   `json:"exam_date"`
	DaysUntilExam *int      `json:"days_until_exam"`
	Reason        *string   `json:"reason"`
	SavedAt       time.Time `json:"saved_at"`
}

type scheduleResponse struct {
	DailyMinutes int       `json:"daily_minutes"`
	StudyDays    []int     `json:"study_days"`
	SavedAt      time.Time `json:"saved_at"`
}

type previousScoreResponse struct {
	Total          int     `json:"total"`
	Math           *int    `json:"math"`
	ReadingWriting *int    `json:"reading_writing"`
	TakenMonth     *string `json:"taken_month"`
}

type backgroundResponse struct {
	MathConfidence           int                    `json:"math_confidence"`
	ReadingWritingConfidence int                    `json:"reading_writing_confidence"`
	PreviousScore            *previousScoreResponse `json:"previous_score"`
	SavedAt                  time.Time              `json:"saved_at"`
}

// onboardingResponse: a step is null until the learner saves it, so "not answered" is never
// confused with a default value.
type onboardingResponse struct {
	Completed   bool                `json:"completed"`
	CompletedAt *time.Time          `json:"completed_at"`
	NextStep    *string             `json:"next_step"`
	Goal        *goalResponse       `json:"goal"`
	Schedule    *scheduleResponse   `json:"schedule"`
	Background  *backgroundResponse `json:"background"`
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intOrNil(v int) *int {
	if v == 0 {
		return nil
	}
	return &v
}

func toResponse(v View) onboardingResponse {
	r := onboardingResponse{Completed: v.CompletedAt != nil, CompletedAt: v.CompletedAt}
	if next := v.NextStep(); next != "" {
		s := string(next)
		r.NextStep = &s
	}
	if v.GoalSavedAt != nil {
		g := goalResponse{
			TargetScore: v.Goal.TargetScore, DaysUntilExam: v.DaysUntilExam(),
			Reason: strOrNil(v.Goal.Reason), SavedAt: *v.GoalSavedAt,
		}
		if v.Goal.ExamDate != nil {
			d := v.Goal.ExamDate.Format(time.DateOnly)
			g.ExamDate = &d
		}
		r.Goal = &g
	}
	if v.ScheduleSavedAt != nil {
		r.Schedule = &scheduleResponse{
			DailyMinutes: v.Schedule.DailyMinutes, StudyDays: v.Schedule.StudyDays, SavedAt: *v.ScheduleSavedAt,
		}
	}
	if v.BackgroundSavedAt != nil {
		b := backgroundResponse{
			MathConfidence: v.Background.MathConfidence, ReadingWritingConfidence: v.Background.ReadingWritingConfidence,
			SavedAt: *v.BackgroundSavedAt,
		}
		if p := v.Background.Previous; p != nil {
			b.PreviousScore = &previousScoreResponse{
				Total: p.Total, Math: intOrNil(p.Math), ReadingWriting: intOrNil(p.ReadingWriting),
			}
			if p.TakenOn != nil {
				m := p.TakenOn.Format("2006-01")
				b.PreviousScore.TakenMonth = &m
			}
		}
		r.Background = &b
	}
	return r
}

func (h *Handler) respond(c *gin.Context, v View, err error) {
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

func (h *Handler) get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), auth.CurrentUser(c).UserID)
	h.respond(c, v, err)
}

type goalRequest struct {
	TargetScore *int    `json:"target_score"`
	ExamDate    *string `json:"exam_date" binding:"omitempty,max=32"`
	Reason      *string `json:"reason" binding:"omitempty,max=64"`
}

func (h *Handler) saveGoal(c *gin.Context) {
	var req goalRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	v, err := h.svc.SaveGoal(c.Request.Context(), auth.CurrentUser(c).UserID, GoalInput(req))
	h.respond(c, v, err)
}

type scheduleRequest struct {
	DailyMinutes *int  `json:"daily_minutes"`
	StudyDays    []int `json:"study_days" binding:"max=14"`
}

func (h *Handler) saveSchedule(c *gin.Context) {
	var req scheduleRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	v, err := h.svc.SaveSchedule(c.Request.Context(), auth.CurrentUser(c).UserID, ScheduleInput(req))
	h.respond(c, v, err)
}

type previousScoreRequest struct {
	Total          *int    `json:"total"`
	Math           *int    `json:"math"`
	ReadingWriting *int    `json:"reading_writing"`
	TakenMonth     *string `json:"taken_month" binding:"omitempty,max=16"`
}

type backgroundRequest struct {
	MathConfidence           *int                  `json:"math_confidence"`
	ReadingWritingConfidence *int                  `json:"reading_writing_confidence"`
	PreviousScore            *previousScoreRequest `json:"previous_score"`
}

func (h *Handler) saveBackground(c *gin.Context) {
	var req backgroundRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	in := BackgroundInput{MathConfidence: req.MathConfidence, ReadingWritingConfidence: req.ReadingWritingConfidence}
	if p := req.PreviousScore; p != nil {
		in.PreviousScore = &PreviousScoreInput{Total: p.Total, Math: p.Math, ReadingWriting: p.ReadingWriting, TakenMonth: p.TakenMonth}
	}
	v, err := h.svc.SaveBackground(c.Request.Context(), auth.CurrentUser(c).UserID, in)
	h.respond(c, v, err)
}

func (h *Handler) complete(c *gin.Context) {
	v, err := h.svc.Complete(c.Request.Context(), auth.CurrentUser(c).UserID)
	h.respond(c, v, err)
}
