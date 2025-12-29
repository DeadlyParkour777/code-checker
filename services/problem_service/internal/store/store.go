package store

import (
	"database/sql"
	"fmt"

	"github.com/DeadlyParkour777/code-checker/services/problem_service/internal/types"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type Store interface {
	CreateProblem(problem *types.Problem) (*types.Problem, error)
	GetProblem(id string) (*types.Problem, error)
	ListProblems() ([]*types.Problem, error)
	CreateTestCase(testCase *types.TestCase) (*types.TestCase, error)
	GetTestCasesByProblemID(problemID string) ([]*types.TestCase, error)
}

type store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreateProblem(problem *types.Problem) (*types.Problem, error) {
	problem.ID = uuid.New().String()

	query := `INSERT INTO problems (id, title, description) VALUES ($1, $2, $3) RETURNING created_at`

	err := s.db.QueryRow(query, problem.ID, problem.Title, problem.Description).Scan(&problem.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create problem: %w", err)
	}

	return problem, nil
}

func (s *store) GetProblem(id string) (*types.Problem, error) {
	problem := &types.Problem{}
	query := `SELECT id, title, description, created_at FROM problems WHERE id = $1`

	err := s.db.QueryRow(query, id).Scan(&problem.ID, &problem.Title, &problem.Description, &problem.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("problem not found")
		}
		return nil, fmt.Errorf("failed to get problem: %w", err)
	}
	return problem, nil
}

func (s *store) ListProblems() ([]*types.Problem, error) {
	var problems []*types.Problem
	query := `SELECT id, title, description, created_at FROM problems`
	rows, err := s.db.Query(query)

	if err != nil {
		return nil, fmt.Errorf("failed to list problems: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		problem := &types.Problem{}
		if err := rows.Scan(&problem.ID, &problem.Title, &problem.Description, &problem.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan problem: %w", err)
		}
		problems = append(problems, problem)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return problems, nil
}

func (s *store) CreateTestCase(testCase *types.TestCase) (*types.TestCase, error) {
	testCase.ID = uuid.New().String()
	query := `INSERT INTO test_cases (id, problem_id, input_data, output_data) VALUES ($1, $2, $3, $4)`

	_, err := s.db.Exec(query, testCase.ID, testCase.ProblemID, testCase.Input, testCase.Output)
	if err != nil {
		return nil, fmt.Errorf("failed to create test case: %w", err)
	}

	return testCase, nil
}

func (s *store) GetTestCasesByProblemID(problemID string) ([]*types.TestCase, error) {
	var testCases []*types.TestCase

	query := `SELECT id, problem_id, input_data, output_data FROM test_cases WHERE problem_id = $1`
	rows, err := s.db.Query(query, problemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get test cases: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		tc := &types.TestCase{}
		if err := rows.Scan(&tc.ID, &tc.ProblemID, &tc.Input, &tc.Output); err != nil {
			return nil, fmt.Errorf("failed to scan test case: %w", err)
		}
		testCases = append(testCases, tc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over test case rows: %w", err)
	}

	return testCases, nil
}
