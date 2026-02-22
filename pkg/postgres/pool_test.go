package postgres

import (
	"testing"
)

func TestConfig_DSN(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{
			name: "basic config with explicit sslmode",
			cfg: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "admin",
				Password: "secret",
				Database: "bankdb",
				SSLMode:  "require",
			},
			want: "postgres://admin:secret@localhost:5432/bankdb?sslmode=require",
		},
		{
			name: "sslmode defaults to require when empty",
			cfg: Config{
				Host:     "localhost",
				Port:     5432,
				User:     "admin",
				Password: "secret",
				Database: "bankdb",
			},
			want: "postgres://admin:secret@localhost:5432/bankdb?sslmode=require",
		},
		{
			name: "custom port and host",
			cfg: Config{
				Host:     "db.example.com",
				Port:     5433,
				User:     "app_user",
				Password: "p@ssw0rd",
				Database: "accounts",
				SSLMode:  "verify-full",
			},
			want: "postgres://app_user:p@ssw0rd@db.example.com:5433/accounts?sslmode=verify-full",
		},
		{
			name: "sslmode prefer",
			cfg: Config{
				Host:     "10.0.0.1",
				Port:     5432,
				User:     "root",
				Password: "toor",
				Database: "transactions",
				SSLMode:  "prefer",
			},
			want: "postgres://root:toor@10.0.0.1:5432/transactions?sslmode=prefer",
		},
		{
			name: "zero port renders as 0",
			cfg: Config{
				Host:     "localhost",
				Port:     0,
				User:     "user",
				Password: "pass",
				Database: "testdb",
			},
			want: "postgres://user:pass@localhost:0/testdb?sslmode=require",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.DSN()
			if got != tt.want {
				t.Errorf("Config.DSN() = %q, want %q", got, tt.want)
			}
		})
	}
}
