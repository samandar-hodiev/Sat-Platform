package onboarding

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
)

type testAPI struct {
	t      *testing.T
	r      *gin.Engine
	tokens *auth.AccessTokens
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	httpx.ConfigureValidator()
	f := newFixture(t)
	tokens, err := auth.NewAccessTokens(strings.Repeat("k", 32), 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(middleware.Errors(slog.New(slog.NewTextHandler(io.Discard, nil))))
	NewHandler(f.svc).RegisterRoutes(r.Group("/api/v1"), auth.Authenticate(tokens))
	return &testAPI{t: t, r: r, tokens: tokens}
}

func (a *testAPI) token(userID string, role auth.Role) string {
	a.t.Helper()
	tok, _, err := a.tokens.Issue(userID, role, "33333333-0000-4000-8000-000000000000")
	if err != nil {
		a.t.Fatal(err)
	}
	return tok
}

type apiResponse struct {
	Data struct {
		Completed   bool    `json:"completed"`
		CompletedAt *string `json:"completed_at"`
		NextStep    *string `json:"next_step"`
		Goal        *struct {
			TargetScore   int     `json:"target_score"`
			ExamDate      *string `json:"exam_date"`
			DaysUntilExam *int    `json:"days_until_exam"`
			Reason        *string `json:"reason"`
		} `json:"goal"`
		Schedule *struct {
			DailyMinutes int   `json:"daily_minutes"`
			StudyDays    []int `json:"study_days"`
		} `json:"schedule"`
		Background *struct {
			MathConfidence           int `json:"math_confidence"`
			ReadingWritingConfidence int `json:"reading_writing_confidence"`
			PreviousScore            *struct {
				Total          int     `json:"total"`
				Math           *int    `json:"math"`
				ReadingWriting *int    `json:"reading_writing"`
				TakenMonth     *string `json:"taken_month"`
			} `json:"previous_score"`
		} `json:"background"`
	} `json:"data"`
	Error struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func (a *testAPI) do(method, path, body, token string) (*httptest.ResponseRecorder, apiResponse) {
	a.t.Helper()
	var buf io.Reader
	if body != "" {
		buf = bytes.NewReader([]byte(body))
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.r.ServeHTTP(w, req)
	var res apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		a.t.Fatalf("invalid JSON %q", w.Body.String())
	}
	return w, res
}

func TestHTTPOnboardingRequiresLearnerToken(t *testing.T) {
	a := newTestAPI(t)
	if w, _ := a.do(http.MethodGet, "/api/v1/me/onboarding", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	w, res := a.do(http.MethodPut, "/api/v1/me/onboarding/goal", `{"target_score":1400}`, a.token(userReady, auth.Role("owner")))
	if w.Code != http.StatusForbidden || res.Error.Code != "FORBIDDEN" {
		t.Fatal(w.Code, res.Error)
	}
}

func TestHTTPEmptyStateIsNull(t *testing.T) {
	a := newTestAPI(t)
	w, _ := a.do(http.MethodGet, "/api/v1/me/onboarding", "", a.token(userReady, auth.RoleLearner))
	var raw struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &raw)
	if w.Code != http.StatusOK || raw.Data["goal"] != nil || raw.Data["schedule"] != nil ||
		raw.Data["background"] != nil || raw.Data["completed"] != false || raw.Data["next_step"] != "goal" {
		t.Fatalf("%d %v", w.Code, raw.Data)
	}
	for _, k := range []string{"goal", "schedule", "background", "completed_at", "next_step"} {
		if _, ok := raw.Data[k]; !ok {
			t.Errorf("key %q missing: null must be sent explicitly", k)
		}
	}
}

func TestHTTPOnboardingFlow(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userReady, auth.RoleLearner)

	w, res := a.do(http.MethodPut, "/api/v1/me/onboarding/goal", `{"target_score":1450,"exam_date":"2027-03-13","reason":null}`, tok)
	if w.Code != http.StatusOK || res.Data.Goal == nil || res.Data.Goal.TargetScore != 1450 ||
		*res.Data.Goal.ExamDate != "2027-03-13" || *res.Data.Goal.DaysUntilExam != 151 || res.Data.Goal.Reason != nil ||
		*res.Data.NextStep != "schedule" {
		t.Fatalf("%d %+v %+v", w.Code, res.Data, res.Error)
	}

	w, res = a.do(http.MethodPost, "/api/v1/me/onboarding/complete", "", tok)
	if w.Code != http.StatusConflict || res.Error.Details["reason"] != "onboarding_incomplete" {
		t.Fatal(w.Code, res.Error)
	}

	w, res = a.do(http.MethodPut, "/api/v1/me/onboarding/schedule", `{"daily_minutes":60,"study_days":[5,1]}`, tok)
	if w.Code != http.StatusOK || res.Data.Schedule.DailyMinutes != 60 || len(res.Data.Schedule.StudyDays) != 2 || res.Data.Schedule.StudyDays[0] != 1 {
		t.Fatalf("%d %+v", w.Code, res.Data.Schedule)
	}

	body := `{"math_confidence":2,"reading_writing_confidence":4,"previous_score":{"total":1150,"math":550,"reading_writing":600,"taken_month":"2026-03"}}`
	w, res = a.do(http.MethodPut, "/api/v1/me/onboarding/background", body, tok)
	p := res.Data.Background.PreviousScore
	if w.Code != http.StatusOK || p == nil || p.Total != 1150 || *p.Math != 550 || *p.TakenMonth != "2026-03" || res.Data.NextStep != nil {
		t.Fatalf("%d %+v", w.Code, res.Data.Background)
	}

	w, res = a.do(http.MethodPost, "/api/v1/me/onboarding/complete", "", tok)
	if w.Code != http.StatusOK || !res.Data.Completed || res.Data.CompletedAt == nil {
		t.Fatal(w.Code, res.Data)
	}
	if w, res = a.do(http.MethodGet, "/api/v1/me/onboarding", "", tok); w.Code != http.StatusOK || !res.Data.Completed {
		t.Fatal(w.Code, res.Data)
	}
}

func TestHTTPPreviousScoreTotalOnly(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userReady, auth.RoleLearner)
	w, res := a.do(http.MethodPut, "/api/v1/me/onboarding/background",
		`{"math_confidence":3,"reading_writing_confidence":3,"previous_score":{"total":1000}}`, tok)
	p := res.Data.Background.PreviousScore
	if w.Code != http.StatusOK || p.Total != 1000 || p.Math != nil || p.ReadingWriting != nil || p.TakenMonth != nil {
		t.Fatalf("%d %+v", w.Code, p)
	}
}

func TestHTTPValidationAndBadJSON(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userReady, auth.RoleLearner)
	w, res := a.do(http.MethodPut, "/api/v1/me/onboarding/goal", `{"exam_date":"yesterday"}`, tok)
	fields, _ := res.Error.Details["fields"].(map[string]any)
	if w.Code != http.StatusUnprocessableEntity || fields["target_score"] == nil || fields["exam_date"] == nil {
		t.Fatal(w.Code, res.Error)
	}
	w, res = a.do(http.MethodPut, "/api/v1/me/onboarding/background",
		`{"math_confidence":3,"reading_writing_confidence":3,"previous_score":{"total":1200,"math":700,"reading_writing":600}}`, tok)
	fields, _ = res.Error.Details["fields"].(map[string]any)
	if w.Code != http.StatusUnprocessableEntity || fields["previous_score.total"] == nil {
		t.Fatal(w.Code, res.Error)
	}
	for _, body := range []string{"{", `{"target_score":"1400"}`, `{"target_score":1400.5}`} {
		if w, _ := a.do(http.MethodPut, "/api/v1/me/onboarding/goal", body, tok); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", body, w.Code)
		}
	}
}

func TestHTTPStepBeforeProfileSetup(t *testing.T) {
	a := newTestAPI(t)
	w, res := a.do(http.MethodPut, "/api/v1/me/onboarding/goal", `{"target_score":1400}`, a.token(userNew, auth.RoleLearner))
	if w.Code != http.StatusConflict || res.Error.Details["reason"] != "profile_setup_required" {
		t.Fatal(w.Code, res.Error)
	}
}
