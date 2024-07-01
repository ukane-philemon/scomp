package main

import (
	"context"
	"flag"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mdb, err := db.NewMongoDB(ctx, dbName, dbURL)
	if err != nil {
		log.Fatalf("mongodb.New error: %v", err)
	}

	resolver := new(graph.Resolver)

	resolver.AdminRepo, err = admin.NewRepository(ctx, mdb)
	if err != nil {
		log.Fatalf("admin.NewRepository error: %v", err)
	}

	resolver.ClassRepo, err = class.NewRepository(ctx, mdb)
	if err != nil {
		log.Fatalf("class.NewRepository error: %v", err)
	}

	resolver.StudentRepo, err = student.NewRepository(ctx, mdb)
	if err != nil {
		log.Fatalf("student.NewRepository error: %v", err)
	}

	resolver.AuthRepo, err = auth.NewRepository()
	if err != nil {
		log.Fatalf("auth.NewRepository error: %v", err)
	}

	// Ensure graceful shutdown by capturing SIGINT and SIGTERM signals.
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-shutdownChan
		resolver.Wait() // wait for asynchronous tasks to finish.
		cancel()

		dbShutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err = db.ShutdownMongoDB(dbShutdownCtx, mdb)
		if err != nil {
			log.Fatalf("db.ShutdownMongoDB error: %v", err)
		}
	}()

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))
	chiMux := chi.NewMux()
	chiMux.Use(middleware.Logger)
	chiMux.Use(graph.AuthMiddleware(resolver.AuthRepo))
	chiMux.Handle("/", playground.Handler("GraphQL playground", "/scomp"))
	chiMux.Handle("/scomp", srv)

	log.Printf("\nSCOMP has started successfully, connect to http://localhost:%s/ for GraphQL playground", port)

	err = http.ListenAndServe(":"+port, chiMux)
	if err != nil && err != http.ErrServerClosed {
		log.Printf("SCOMP shutdown error: %v", err)
	} else {
		log.Println("SCOMP shutdown successfully...")
	}
}
