module content-service

go 1.25.1

require (
	github.com/Anabol1ks/Forklore/pkg/pb v0.0.0
	github.com/Anabol1ks/Forklore/pkg/utils v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.79.2
	gorm.io/gorm v1.25.10
)

require (
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.6.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gorm.io/driver/postgres v1.6.0 // indirect
)

replace github.com/Anabol1ks/Forklore/pkg/utils => ../pkg/utils

replace github.com/Anabol1ks/Forklore/pkg/pb => ../pkg/pb
