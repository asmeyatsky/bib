module github.com/bibbank/bib/services/deposit-service

go 1.24

require (
	github.com/bibbank/bib/pkg/events v0.0.0
	github.com/bibbank/bib/pkg/kafka v0.0.0
	github.com/bibbank/bib/pkg/money v0.0.0
	github.com/bibbank/bib/pkg/observability v0.0.0
	github.com/bibbank/bib/pkg/postgres v0.0.0
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.10.0
	google.golang.org/grpc v1.68.1
)

replace (
	github.com/bibbank/bib/pkg/events => ../../pkg/events
	github.com/bibbank/bib/pkg/kafka => ../../pkg/kafka
	github.com/bibbank/bib/pkg/money => ../../pkg/money
	github.com/bibbank/bib/pkg/observability => ../../pkg/observability
	github.com/bibbank/bib/pkg/postgres => ../../pkg/postgres
)
