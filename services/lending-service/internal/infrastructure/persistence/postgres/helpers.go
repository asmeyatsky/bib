package postgres

import "time"

// pgTime wraps time.Time for pgx scanning.
type pgTime struct {
	t time.Time
}

func (p *pgTime) Scan(src interface{}) error {
	switch v := src.(type) {
	case time.Time:
		p.t = v
		return nil
	default:
		return nil
	}
}

func (p pgTime) Time() time.Time {
	return p.t
}
