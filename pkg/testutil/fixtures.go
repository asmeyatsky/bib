package testutil

import (
	"github.com/google/uuid"
)

// Fixed UUIDs for deterministic testing
var (
	TestUserID1   = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	TestUserID2   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	TestTenantID  = uuid.MustParse("00000000-0000-0000-0000-000000000010")
	TestAccountID = uuid.MustParse("00000000-0000-0000-0000-000000000020")
)
