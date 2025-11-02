package registry

import (
	"database/sql"
	"log"

	pb "github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/pb"
	"github.com/yhonda-ohishi-pub-dev/etc_meisai_scraper/src/services"
	"google.golang.org/grpc"
)

// ServiceRegistry holds all etc_meisai_scraper gRPC service implementations
type ServiceRegistry struct {
	DownloadService pb.DownloadServiceServer
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(db *sql.DB, logger *log.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		DownloadService: services.NewDownloadServiceGRPC(db, logger),
	}
}

// RegisterAll registers all services to the gRPC server
func (r *ServiceRegistry) RegisterAll(server *grpc.Server) {
	if r.DownloadService != nil {
		pb.RegisterDownloadServiceServer(server, r.DownloadService)
		if log.Default() != nil {
			log.Println("Registered: DownloadService")
		}
	}
}

// Register is a convenience function that creates a registry and registers all services
func Register(server *grpc.Server, db *sql.DB, logger *log.Logger) *ServiceRegistry {
	registry := NewServiceRegistry(db, logger)
	if registry == nil {
		log.Println("Warning: etc_meisai_scraper registry not available")
		return nil
	}

	registry.RegisterAll(server)
	return registry
}
