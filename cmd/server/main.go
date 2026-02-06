package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/api"
	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/store"
	"github.com/erazemk/skladisce/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: skladisce <init|serve>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: skladisce <init|serve>\n", os.Args[1])
		os.Exit(1)
	}
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dbPath := fs.String("db", "skladisce.sqlite3", "path to SQLite database file")
	fs.Parse(args)

	if _, err := os.Stat(*dbPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: database file %s already exists\n", *dbPath)
		os.Exit(1)
	}

	database, password, err := initDatabase(*dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	database.Close()

	fmt.Printf("Database created: %s\n", *dbPath)
	fmt.Println("Schema initialized.")
	fmt.Println()
	fmt.Println("Admin account created:")
	fmt.Printf("  Username: admin\n")
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()
	fmt.Println("Save this password — it cannot be recovered.")
	fmt.Println("The admin can change it after logging in.")
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dbPath := fs.String("db", "skladisce.sqlite3", "path to SQLite database file")
	addr := fs.String("addr", ":8080", "listen address")
	jwtSecret := fs.String("jwt-secret", "", "JWT signing key (auto-generated if empty)")
	fs.Parse(args)

	// Auto-generate JWT secret if not provided.
	if *jwtSecret == "" {
		secret, err := generatePassword(32)
		if err != nil {
			log.Fatalf("Failed to generate JWT secret: %v", err)
		}
		*jwtSecret = secret
		log.Println("JWT secret auto-generated (tokens will be invalidated on restart)")
	}

	// Check if DB exists, auto-init if not.
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		database, password, err := initDatabase(*dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
		database.Close()

		fmt.Printf("Database created: %s\n", *dbPath)
		fmt.Println("Schema initialized.")
		fmt.Println()
		fmt.Println("Admin account created:")
		fmt.Printf("  Username: admin\n")
		fmt.Printf("  Password: %s\n", password)
		fmt.Println()
		fmt.Println("Save this password — it cannot be recovered.")
		fmt.Println("The admin can change it after logging in.")
		fmt.Println()
	}

	// Open database.
	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	// Run migrations (idempotent).
	if err := db.Migrate(database); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Set up routers.
	apiRouter := api.NewRouter(database, *jwtSecret)
	webRouter, err := web.NewRouter(database, *jwtSecret)
	if err != nil {
		log.Fatalf("Failed to set up web router: %v", err)
	}

	// Combine: API routes take priority, web routes handle the rest.
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	handler := api.LoggingMiddleware(mux)

	fmt.Printf("Server listening on %s\n", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// initDatabase creates a new database, runs migrations, and creates the admin user.
func initDatabase(path string) (*sql.DB, string, error) {
	database, err := db.Open(path)
	if err != nil {
		return nil, "", fmt.Errorf("opening database: %w", err)
	}

	if err := db.Migrate(database); err != nil {
		database.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("running migrations: %w", err)
	}

	password, err := generatePassword(16)
	if err != nil {
		database.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("generating password: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		database.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("hashing password: %w", err)
	}

	ctx := context.Background()
	_, err = store.CreateUser(ctx, database, "admin", string(hash), "admin")
	if err != nil {
		database.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("creating admin user: %w", err)
	}

	return database, password, nil
}

// generatePassword creates a random password of the given length.
func generatePassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
}
