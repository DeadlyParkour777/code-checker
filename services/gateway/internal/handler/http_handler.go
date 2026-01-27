package handler

import (
	"bytes"
	"context"
	"embed"
	"io"
	"net/http"
	"strings"

	authpb "github.com/DeadlyParkour777/code-checker/pkg/auth"
	problempb "github.com/DeadlyParkour777/code-checker/pkg/problem"
	resultpb "github.com/DeadlyParkour777/code-checker/pkg/result"
	submissionpb "github.com/DeadlyParkour777/code-checker/pkg/submission"
	"github.com/DeadlyParkour777/code-checker/pkg/utils"
	"github.com/DeadlyParkour777/code-checker/services/gateway/internal/cache"
	"github.com/DeadlyParkour777/code-checker/services/gateway/internal/types"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-playground/validator/v10"
	httpSwagger "github.com/swaggo/http-swagger"
)

//go:embed openapi.yaml
var openApiSpec embed.FS

type userCtxKey string

const userIDKey = userCtxKey("userID")
const userRoleKey = userCtxKey("userRole")

type Handler struct {
	authClient       authpb.AuthServiceClient
	problemClient    problempb.ProblemServiceClient
	submissionClient submissionpb.SubmissionServiceClient
	resultClient     resultpb.ResultServiceClient
	jwtCache         cache.JWTCache
	validator        *validator.Validate
}

func NewHandler(
	authClient authpb.AuthServiceClient,
	problemClient problempb.ProblemServiceClient,
	submissionClient submissionpb.SubmissionServiceClient,
	resultClient resultpb.ResultServiceClient,
	jwtCache cache.JWTCache,
) *Handler {
	return &Handler{
		authClient:       authClient,
		problemClient:    problemClient,
		submissionClient: submissionClient,
		resultClient:     resultClient,
		jwtCache:         jwtCache,
		validator:        validator.New(),
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		data, err := openApiSpec.ReadFile("openapi.yaml")
		if err != nil {
			http.Error(w, "Spec not found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-yaml")
		w.Write(data)
	})

	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/openapi.yaml"),
	))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, http.StatusOK, "Gateway is running")
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.handleRegister)
		r.Post("/login", h.handleLogin)
	})

	r.Route("/problems", func(r chi.Router) {
		r.Get("/", h.handleListProblems)
		r.Get("/{problemID}", h.handleGetProblem)
	})

	r.Group(func(r chi.Router) {
		r.Use(h.AuthMiddleware)

		r.Group(func(r chi.Router) {
			r.Use(h.AdminOnlyMiddleware)
			r.Post("/problems", h.handleCreateProblem)
			r.Post("/problems/{problemID}/testcases", h.handleCreateTestCase)
		})

		r.Route("/submissions", func(r chi.Router) {
			r.Post("/", h.handleCreateSubmission)
			r.Get("/history", h.handleGetUserSubmissions)
		})
	})

	return r
}

func (h *Handler) AdminOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(userRoleKey).(string)
		if !ok {
			utils.WriteError(w, http.StatusInternalServerError, "Could not retrieve user role")
			return
		}

		if role != "admin" {
			utils.WriteError(w, http.StatusForbidden, "Forbidden: Admins only")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.WriteError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		parts := strings.Split(authHeader, "Bearer ")
		if len(parts) != 2 {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}
		token := parts[1]

		resp, err := h.authClient.ValidateToken(r.Context(), &authpb.ValidateRequest{Token: token})
		if err != nil || !resp.GetValid() {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, resp.GetUserId())
		ctx = context.WithValue(ctx, userRoleKey, resp.GetRole())
		next.ServeHTTP(w, r.WithContext(ctx))
	})

}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req types.RegisterRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authClient.Register(r.Context(), &authpb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req types.LoginRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authClient.Login(r.Context(), &authpb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleCreateProblem(w http.ResponseWriter, r *http.Request) {
	var req types.CreateProblemRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.problemClient.CreateProblem(r.Context(), &problempb.CreateProblemRequest{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) handleListProblems(w http.ResponseWriter, r *http.Request) {
	resp, err := h.problemClient.ListProblems(r.Context(), &problempb.ListProblemsRequest{})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, resp.Problems)
}

func (h *Handler) handleGetProblem(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemID")
	if problemID == "" {
		utils.WriteError(w, http.StatusBadRequest, "Problem ID is required")
		return
	}

	resp, err := h.problemClient.GetProblem(r.Context(), &problempb.GetProblemRequest{Id: problemID})
	if err != nil {
		utils.WriteError(w, http.StatusNotFound, "Problem not found")
		return
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleCreateSubmission(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Failed to parse multipart form: "+err.Error())
		return
	}

	problemID := r.FormValue("problem_id")
	language := r.FormValue("language")
	if problemID == "" || language == "" {
		utils.WriteError(w, http.StatusBadRequest, "problem_id and language are required fields")
		return
	}

	file, _, err := r.FormFile("code_file")
	if err != nil {
		utils.WriteError(w, http.StatusBadRequest, "Code file with key 'code_file' is required")
		return
	}
	defer file.Close()

	codeData, err := io.ReadAll(file)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to read code file")
		return
	}

	stream, err := h.submissionClient.CreateSubmission(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to create submission stream: "+err.Error())
		return
	}

	info := &submissionpb.SubmissionInfo{
		UserId:    userID,
		ProblemId: problemID,
		Language:  language,
	}
	if err := stream.Send(&submissionpb.CreateSubmissionRequest{Data: &submissionpb.CreateSubmissionRequest_Info{Info: info}}); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to send submission info: "+err.Error())
		return
	}

	reader := bytes.NewReader(codeData)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to read code chunk from buffer: "+err.Error())
			return
		}

		req := &submissionpb.CreateSubmissionRequest{
			Data: &submissionpb.CreateSubmissionRequest_ChunkData{ChunkData: buffer[:n]},
		}
		if err := stream.Send(req); err != nil {
			utils.WriteError(w, http.StatusInternalServerError, "Failed to send code chunk: "+err.Error())
			return
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, "Failed to receive submission response: "+err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusAccepted, resp)
}

func (h *Handler) handleGetUserSubmissions(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)

	resp, err := h.resultClient.GetUserSubmissions(r.Context(), &resultpb.GetUserSubmissionsRequest{UserId: userID})
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, resp.Submissions)
}

func (h *Handler) handleCreateTestCase(w http.ResponseWriter, r *http.Request) {
	problemID := chi.URLParam(r, "problemID")
	if problemID == "" {
		utils.WriteError(w, http.StatusBadRequest, "Problem ID is required in URL")
		return
	}

	var req types.CreateTestCaseRequest
	if err := utils.ParseJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.validator.Struct(req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	grpcReq := &problempb.CreateTestCaseRequest{
		ProblemId:  problemID,
		InputData:  req.InputData,
		OutputData: req.OutputData,
	}

	resp, err := h.problemClient.CreateTestCase(r.Context(), grpcReq)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusCreated, resp)
}
