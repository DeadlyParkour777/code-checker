package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	dbURL := getPostgresDSN()

	migrationsPaths := []string{
		"file:///migrations/migrate",
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: ./migrate [up|down]")
	}

	command := os.Args[1]

	for _, path := range migrationsPaths {
		log.Printf("Running migration command: %s", command)

		m, err := migrate.New(path, dbURL)
		if err != nil {
			log.Fatalf("Cannot create migrate instance: %v", err)
		}

		var errMigration error
		switch command {
		case "up":
			errMigration = m.Up()
		case "down":
			errMigration = m.Down()
		default:
			log.Fatalf("Unknown command: %s", command)
		}

		if errMigration != nil && errMigration != migrate.ErrNoChange {
			log.Fatalf("Migration failed: %v", errMigration)
		}

		log.Println("Migration finished successfully!")
	}

}

func getPostgresDSN() string {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	// port := os.Getenv("DB_PORT")
	dbname := os.Getenv("DB_NAME")
	internalPort := "5432"

	return "postgres://" + user + ":" + password + "@" + host + ":" + internalPort + "/" + dbname + "?sslmode=disable"
}
