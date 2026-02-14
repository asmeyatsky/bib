module github.com/bibbank/bib/gateway

go 1.23

require (
	github.com/bibbank/bib/pkg/auth v0.0.0
	github.com/bibbank/bib/pkg/observability v0.0.0
	github.com/google/uuid v1.6.0
	google.golang.org/grpc v1.68.1
)

replace (
	github.com/bibbank/bib/pkg/auth => ../pkg/auth
	github.com/bibbank/bib/pkg/observability => ../pkg/observability
)
