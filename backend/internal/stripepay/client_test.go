package stripepay

import "testing"

func TestIsTestMode(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"sk_test_abc123", true},
		{"sk_live_abc123", false},
		{"", false},
	}
	for _, c := range cases {
		got := Config{SecretKey: c.key}.IsTestMode()
		if got != c.want {
			t.Errorf("key=%q: got %v want %v", c.key, got, c.want)
		}
	}
}
