package store

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/DeadlyParkour777/code-checker/services/problem_service/internal/types"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDB *sql.DB
var testContainer *postgres.PostgresContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic(err)
	}

	metaCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	port, err := container.MappedPort(metaCtx, "5432/tcp")
	if err != nil {
		_ = container.Terminate(ctx)
		panic(err)
	}

	host := "127.0.0.1"
	connStr := fmt.Sprintf("postgres://postgres:postgres@%s:%s/testdb?sslmode=disable&connect_timeout=5", host, port.Port())

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		_ = container.Terminate(ctx)
		panic(err)
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(1 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Second)

	if err := waitForTCP(host, port.Port(), 20*time.Second); err != nil {
		_ = db.Close()
		_ = container.Terminate(ctx)
		panic(err)
	}

	if err := waitForDB(db, 30*time.Second); err != nil {
		dumpContainerLogs(ctx, container)
		_ = db.Close()
		_ = container.Terminate(ctx)
		panic(err)
	}

	if err := createSchema(db); err != nil {
		dumpContainerLogs(ctx, container)
		_ = db.Close()
		_ = container.Terminate(ctx)
		panic(err)
	}

	testDB = db
	testContainer = container

	code := m.Run()

	_ = testDB.Close()
	_ = testContainer.Terminate(ctx)

	os.Exit(code)
}

func dumpContainerLogs(ctx context.Context, container *postgres.PostgresContainer) {
	logs, err := container.Logs(ctx)
	if err != nil {
		return
	}
	defer logs.Close()
	_, _ = io.ReadAll(logs)
}

func waitForDB(db *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		pingCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(200 * time.Millisecond)
	}
	return lastErr
}

func waitForTCP(host, port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 1*time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return lastErr
}

func createSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS problems (
			id UUID PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS test_cases (
			id UUID PRIMARY KEY,
			problem_id UUID NOT NULL REFERENCES problems(id) ON DELETE CASCADE,
			input_data TEXT NOT NULL,
			output_data TEXT NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, stmt := range statements {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := db.ExecContext(ctx, stmt)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}

func resetDB(t *testing.T) {
	t.Helper()
	if _, err := testDB.Exec(`TRUNCATE TABLE test_cases, problems RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("failed to reset db: %v", err)
	}
}

func TestStore_CreateAndGetProblem(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	created, err := s.CreateProblem(&types.Problem{Title: "Two Sum", Description: "Find indices"})
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected id to be set")
	}

	fetched, err := s.GetProblem(created.ID)
	if err != nil {
		t.Fatalf("get problem: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("unexpected id: %s", fetched.ID)
	}
	if fetched.Title != "Two Sum" {
		t.Fatalf("unexpected title: %s", fetched.Title)
	}
}

func TestStore_GetProblem_NotFound(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.GetProblem("00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatalf("expected not found error")
	}
}

func TestStore_ListProblems(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.CreateProblem(&types.Problem{Title: "A", Description: "B"})
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}
	_, err = s.CreateProblem(&types.Problem{Title: "C", Description: "D"})
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}

	problems, err := s.ListProblems()
	if err != nil {
		t.Fatalf("list problems: %v", err)
	}
	if len(problems) != 2 {
		t.Fatalf("expected 2 problems, got %d", len(problems))
	}
}

func TestStore_CreateAndGetTestCases(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	problem, err := s.CreateProblem(&types.Problem{Title: "A", Description: "B"})
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}

	case1, err := s.CreateTestCase(&types.TestCase{ProblemID: problem.ID, Input: "1 2", Output: "3"})
	if err != nil {
		t.Fatalf("create test case: %v", err)
	}
	if case1.ID == "" {
		t.Fatalf("expected test case id to be set")
	}

	_, err = s.CreateTestCase(&types.TestCase{ProblemID: problem.ID, Input: "2 2", Output: "4"})
	if err != nil {
		t.Fatalf("create test case: %v", err)
	}

	cases, err := s.GetTestCasesByProblemID(problem.ID)
	if err != nil {
		t.Fatalf("get test cases: %v", err)
	}
	if len(cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(cases))
	}
}

func TestStore_CreateTestCase_InvalidProblem(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.CreateTestCase(&types.TestCase{
		ProblemID: "00000000-0000-0000-0000-000000000000",
		Input:     "1",
		Output:    "1",
	})
	if err == nil {
		t.Fatalf("expected error for invalid problem_id")
	}
}

func TestStore_GetTestCases_Empty(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	problem, err := s.CreateProblem(&types.Problem{Title: "A", Description: "B"})
	if err != nil {
		t.Fatalf("create problem: %v", err)
	}

	cases, err := s.GetTestCasesByProblemID(problem.ID)
	if err != nil {
		t.Fatalf("get test cases: %v", err)
	}
	if len(cases) != 0 {
		t.Fatalf("expected 0 test cases, got %d", len(cases))
	}
}
