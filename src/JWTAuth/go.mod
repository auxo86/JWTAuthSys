module JWTAuth

go 1.17

require (
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/go-redis/redis/v8 v8.2.3
	github.com/jackc/pgx/v4 v4.10.1
	github.com/joho/godotenv v1.3.0
	tsgh.edu.tw/UsrAuthForWebapi v0.0.0-00010101000000-000000000000
)

require (
	github.com/cespare/xxhash/v2 v2.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.8.0 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.6 // indirect
	github.com/jackc/pgservicefile v0.0.0-20200714003250-2b9c44734f2b // indirect
	github.com/jackc/pgtype v1.6.2 // indirect
	github.com/jackc/puddle v1.1.3 // indirect
	go.opentelemetry.io/otel v0.11.0 // indirect
	golang.org/x/crypto v0.0.0-20200622213623-75b288015ac9 // indirect
	golang.org/x/text v0.3.3 // indirect
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543 // indirect
)

replace tsgh.edu.tw/UsrAuthForWebapi => ../UsrAuthForWebapi
