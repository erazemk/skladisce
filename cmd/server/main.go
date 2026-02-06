package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/erazemk/skladisce/internal/api"
	"github.com/erazemk/skladisce/internal/db"
	"github.com/erazemk/skladisce/internal/store"
	"github.com/erazemk/skladisce/internal/web"
)

const logFile = "skladisce.log"

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

// setupLogger configures slog to write to both stdout and the log file.
func setupLogger() (*os.File, error) {
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}

	w := io.MultiWriter(os.Stdout, f)
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	return f, nil
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dbPath := fs.String("db", "skladisce.sqlite3", "path to SQLite database file")
	adminUser := fs.String("admin", "admin", "admin account username")
	fs.Parse(args)

	if _, err := os.Stat(*dbPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: database file %s already exists\n", *dbPath)
		os.Exit(1)
	}

	database, password, err := initDatabase(*dbPath, *adminUser)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	database.Close()

	printInitResult(*dbPath, *adminUser, password)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	dbPath := fs.String("db", "skladisce.sqlite3", "path to SQLite database file")
	addr := fs.String("addr", ":8080", "listen address")
	jwtSecret := fs.String("jwt-secret", "", "JWT signing key (auto-generated if empty)")
	adminUser := fs.String("admin", "admin", "admin account username (used if DB is auto-initialized)")
	fs.Parse(args)

	// Set up structured logging to stdout + file.
	logf, err := setupLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer logf.Close()

	// Auto-generate JWT secret if not provided.
	if *jwtSecret == "" {
		secret, err := generatePassword(32)
		if err != nil {
			slog.Error("failed to generate JWT secret", "error", err)
			os.Exit(1)
		}
		*jwtSecret = secret
		slog.Warn("JWT secret auto-generated, tokens will be invalidated on restart")
	}

	// Check if DB exists, auto-init if not.
	if _, err := os.Stat(*dbPath); os.IsNotExist(err) {
		database, password, err := initDatabase(*dbPath, *adminUser)
		if err != nil {
			slog.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}
		database.Close()

		printInitResult(*dbPath, *adminUser, password)
		fmt.Println()
	}

	// Open database.
	database, err := db.Open(*dbPath)
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Run migrations (idempotent).
	if err := db.Migrate(database); err != nil {
		slog.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	slog.Info("database ready", "path", *dbPath)

	// Set up routers.
	apiRouter := api.NewRouter(database, *jwtSecret)
	webRouter, err := web.NewRouter(database, *jwtSecret)
	if err != nil {
		slog.Error("failed to set up web router", "error", err)
		os.Exit(1)
	}

	// Combine: API routes take priority, web routes handle the rest.
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/", webRouter)

	handler := api.LoggingMiddleware(mux)

	server := &http.Server{
		Addr:    *addr,
		Handler: handler,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-quit
		slog.Info("shutdown signal received", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("server forced to shutdown", "error", err)
		}
	}()

	slog.Info("server started", "addr", *addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}

	slog.Info("server stopped, closing database")
}

// initDatabase creates a new database, runs migrations, and creates the admin user.
func initDatabase(path, adminUsername string) (*sql.DB, string, error) {
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
	_, err = store.CreateUser(ctx, database, adminUsername, string(hash), "admin")
	if err != nil {
		database.Close()
		os.Remove(path)
		return nil, "", fmt.Errorf("creating admin user: %w", err)
	}

	return database, password, nil
}

// printInitResult prints the database initialization result to stdout.
func printInitResult(dbPath, username, password string) {
	fmt.Printf("Database created: %s\n", dbPath)
	fmt.Println("Schema initialized.")
	fmt.Println()
	fmt.Println("Admin account created:")
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Password: %s\n", password)
	fmt.Println()
	fmt.Println("Save this password — it cannot be recovered.")
	fmt.Println("The admin can change it after logging in.")
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
