package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/httprate"
	"github.com/ukane-philemon/scomp/graph"
	"github.com/ukane-philemon/scomp/internal/admin"
	"github.com/ukane-philemon/scomp/internal/auth"
	"github.com/ukane-philemon/scomp/internal/class"
	"github.com/ukane-philemon/scomp/internal/db"
	"github.com/ukane-philemon/scomp/internal/student"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is not set")
	}

	var isDevMode bool
	flag.BoolVar(&isDevMode, "dev", false, "Run server in development mode")
	flag.Parse()

	var dbName = "scomp"
	if isDevMode {
		dbName = "dev_scomp"
	}

	serverError := runServer(port, dbName, dbURL)
	if serverError != nil {
		log.Fatalf("SCOMP shutdown error: %v", serverError)
	}

	log.Println("SCOMP shutdown successfully...")
}

// runServer prepares and starts the server.
func runServer(port, dbName, dbURL string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mdb, err := db.NewMongoDB(ctx, dbName, dbURL)
	if err != nil {
		return fmt.Errorf("mongodb.New error: %v", err)
	}

	resolver := new(graph.Resolver)

	resolver.AdminRepository, err = admin.NewRepository(ctx, mdb)
	if err != nil {
		return fmt.Errorf("admin.NewRepository error: %v", err)
	}

	resolver.ClassRepository, err = class.NewRepository(ctx, mdb)
	if err != nil {
		return fmt.Errorf("class.NewRepository error: %v", err)
	}

	resolver.StudentRepository, err = student.NewRepository(ctx, mdb)
	if err != nil {
		return fmt.Errorf("student.NewRepository error: %v", err)
	}

	resolver.AuthenticationRepository, err = auth.NewRepository()
	if err != nil {
		return fmt.Errorf("auth.NewRepository error: %v", err)
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	chiMux := chi.NewMux()
	chiMux.Use(middleware.Logger)
	chiMux.Use(middleware.Recoverer)
	chiMux.Use(httprate.LimitByIP(20, 1*time.Minute))
	chiMux.Use(graph.AuthMiddleware(resolver.AuthenticationRepository))
	chiMux.Handle("/", playground.Handler("GraphQL playground", "/scomp"))
	chiMux.Handle("/scomp", srv)

	s := http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      chiMux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Ensure graceful shutdown by capturing SIGINT and SIGTERM signals.
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdownChan
		resolver.Wait() // wait for asynchronous tasks to finish.
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		err = s.Shutdown(shutdownCtx)
		if err != nil {
			log.Printf("server.Shutdown error: %v", err)
		}

		dbShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = db.ShutdownMongoDB(dbShutdownCtx, mdb)
		if err != nil {
			log.Printf("db.ShutdownMongoDB error: %v", err)
		}
	}()

	// Listen.
	var serverError error
	go func() {
		err = s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverError = fmt.Errorf("s.ListentAndServe error: %v", err)
		}
	}()

	log.Printf("SCOMP has started successfully, connect to http://localhost:%s/ for GraphQL playground", port)

	// Block until context is cancelled.
	<-ctx.Done()

	return serverError
}
