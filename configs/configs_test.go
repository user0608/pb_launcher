package configs

import "testing"

func TestValidatePublicURL(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, url string
		valid     bool
	}{
		{"https origin", "https://pb.test", true},
		{"custom port", "http://pb.test:7080", true},
		{"wrong domain", "https://other.test", false},
		{"relative", "/test", false},
		{"credentials", "https://user:pass@pb.test", false},
		{"path", "https://pb.test/path", false},
		{"query", "https://pb.test?x=1", false},
		{"fragment", "https://pb.test#x", false},
		{"wrong scheme", "ftp://pb.test", false},
		{"bad port", "https://pb.test:99999", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := validatePublicURL(tc.url, "pb.test"); (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
}
