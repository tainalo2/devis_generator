package db

import "testing"

func TestToMigrateURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"postgres://u:p@localhost:5432/db?sslmode=disable", "pgx5://u:p@localhost:5432/db?sslmode=disable"},
		{"postgresql://u:p@localhost/db", "pgx5://u:p@localhost/db"},
		{"pgx5://already", "pgx5://already"},
	}
	for _, tc := range cases {
		if got := toMigrateURL(tc.in); got != tc.want {
			t.Fatalf("toMigrateURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
