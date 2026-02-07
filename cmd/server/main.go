package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"flag"
	"fmt"
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

// levelRouter is a slog.Handler that routes INFO/WARN to stdout and ERROR+ to stderr.
type levelRouter struct {
	stdout slog.Handler
	stderr slog.Handler
}

func (lr *levelRouter) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (lr *levelRouter) Handle(ctx context.Context, r slog.Record) error {
	if r.Level >= slog.LevelError {
		return lr.stderr.Handle(ctx, r)
	}
	return lr.stdout.Handle(ctx, r)
}

func (lr *levelRouter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &levelRouter{
		stdout: lr.stdout.WithAttrs(attrs),
		stderr: lr.stderr.WithAttrs(attrs),
	}
}

func (lr *levelRouter) WithGroup(name string) slog.Handler {
	return &levelRouter{
		stdout: lr.stdout.WithGroup(name),
		stderr: lr.stderr.WithGroup(name),
	}
}

func setupLogger() {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	handler := &levelRouter{
		stdout: slog.NewTextHandler(os.Stdout, opts),
		stderr: slog.NewTextHandler(os.Stderr, opts),
	}
	slog.SetDefault(slog.New(handler))
}

func main() {
	fs := flag.NewFlagSet("skladisce", flag.ContinueOnError)

	var dbPath string
	fs.StringVar(&dbPath, "db", "skladisce.sqlite3", "")
	fs.StringVar(&dbPath, "d", "skladisce.sqlite3", "")

	var addr string
	fs.StringVar(&addr, "addr", ":8080", "")
	fs.StringVar(&addr, "a", ":8080", "")

	var adminUser string
	fs.StringVar(&adminUser, "username", "admin", "")
	fs.StringVar(&adminUser, "u", "admin", "")

	fs.Usage = func() {
		fmt.Fprint(os.Stdout, `Usage: skladisce [flags]

Flags:
  -d, -db <path>          SQLite database path (default: skladisce.sqlite3)
  -a, -addr <host:port>   listen address (default: :8080)
  -u, -username <name>    admin username on first run (default: admin)
  -h, -help               show this help and exit
`)
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected argument: %s\n", fs.Arg(0))
		fs.Usage()
		os.Exit(1)
	}

	// Set up structured logging: INFO/WARN → stdout, ERROR → stderr.
	setupLogger()

	// Check if DB exists, auto-init if not.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		database, password, err := initDatabase(dbPath, adminUser)
		if err != nil {
			slog.Error("failed to initialize database", "error", err)
			os.Exit(1)
		}
		database.Close()

		printInitResult(dbPath, adminUser, password)
		fmt.Println()
	}

	// Open database.
	database, err := db.Open(dbPath)
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

	slog.Info("database ready", "path", dbPath)

	// Load JWT secret from database (auto-generated on first run).
	jwtSecret, err := store.GetJWTSecret(context.Background(), database)
	if err != nil {
		slog.Error("failed to get JWT secret", "error", err)
		os.Exit(1)
	}

	// Set up routers.
	apiRouter := api.NewRouter(database, jwtSecret)
	webRouter, err := web.NewRouter(database, jwtSecret)
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
		Addr:    addr,
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

	slog.Info("server started", "addr", addr)
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
