package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/chi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"fis-playground/internal/handlers"
	"fis-playground/internal/repository"
)

var chiLambda *chiadapter.ChiLambda

// init initializes the Chi router and Lambda adapter
func init() {
	log.Println("Initializing Chi router...")
	
	// Create context for initialization
	ctx := context.Background()
	
	// Initialize repository dependencies
	clientManager, err := repository.NewClientManager(ctx)
	if err != nil {
		log.Fatalf("Failed to create client manager: %v", err)
	}

	repo := repository.NewDynamoDBRepositoryFromManager(clientManager)
	itemHandler := handlers.NewItemHandler(repo)

	// Create Chi router
	r := chi.NewRouter()

	// Add middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Add CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Requested-With"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check endpoint
	r.Get("/health", itemHandler.HealthCheck)
	r.Get("/", itemHandler.HealthCheck) // Root path health check

	// API routes
	r.Route("/items", func(r chi.Router) {
		r.Get("/", itemHandler.ListItems)
		r.Post("/", itemHandler.CreateItem)
		
		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", itemHandler.GetItem)
			r.Put("/", itemHandler.UpdateItem)
			r.Delete("/", itemHandler.DeleteItem)
		})
	})

	// Initialize the Chi Lambda adapter
	chiLambda = chiadapter.New(r)
}

// Handler is the main Lambda handler function
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return chiLambda.ProxyWithContext(ctx, req)
}

func main() {
	lambda.Start(Handler)
}
