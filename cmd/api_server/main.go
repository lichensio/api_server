package main

import (
	"fmt"
	repo "github.com/lichensio/api_server/db/repo"
	lhttp "github.com/lichensio/api_server/pkg/api/http"
	"github.com/lichensio/api_server/pkg/api/service"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {

	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.InfoLevel)

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SSLMODE"),
	)
	dbname, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// Setup repository
	nrepo := repo.NewRepositoryWithDB(dbname)
	if err != nil {
		log.Fatalf("failed to create repository: %v", err)
	}

	// Setup service
	serv := service.NewEmployeeService(nrepo)
	services := &lhttp.Service{
		EmployeeService: serv,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8070" // Default to port 8070 if not specified
	}

	r := lhttp.NewRouter(services)

	// Middlewares
	// r.Use(middleware.RequestID)
	// r.Use(middleware.RealIP)
	// r.Use(lmiddleware.LoggingMiddleware)
	// r.Use(middleware.Recoverer)
	// r.Use(lmiddleware.AuthMiddleware) // Custom Auth middleware

	log.Info("Starting server on ", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
