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

	"github.com/DeadlyParkour777/code-checker/services/auth_service/internal/types"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testDB *sql.DB
var testContainer *postgres.PostgresContainer

func TestMain(m *testing.M) {
	_ = os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")

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
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		role VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`)
	return err
}

func resetDB(t *testing.T) {
	t.Helper()
	if _, err := testDB.Exec(`TRUNCATE TABLE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("failed to reset db: %v", err)
	}
}

func TestStore_CreateAndGetUserByUsername(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	created, err := s.CreateUser(&types.User{Username: "alice", Password: "hash", Role: "user"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected id to be set")
	}

	fetched, err := s.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if fetched.ID != created.ID {
		t.Fatalf("unexpected id: %s", fetched.ID)
	}
	if fetched.Role != "user" {
		t.Fatalf("unexpected role: %s", fetched.Role)
	}
}

func TestStore_GetUserByUsername_NotFound(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.GetUserByUsername("missing")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestStore_GetUserByID(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	created, err := s.CreateUser(&types.User{Username: "bob", Password: "hash", Role: "admin"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	fetched, err := s.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if fetched.Username != "bob" {
		t.Fatalf("unexpected username: %s", fetched.Username)
	}
}

func TestStore_GetUserByID_NotFound(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.GetUserByID("00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestStore_CreateUser_UniqueUsername(t *testing.T) {
	resetDB(t)

	s := NewStore(testDB)

	_, err := s.CreateUser(&types.User{Username: "alice", Password: "hash", Role: "user"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	_, err = s.CreateUser(&types.User{Username: "alice", Password: "hash", Role: "user"})
	if err == nil {
		t.Fatalf("expected unique constraint error")
	}
}
