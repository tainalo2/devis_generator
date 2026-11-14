package httpapi

// testOptions returns a minimal Options struct suitable for tests,
// with safe defaults (including a non-empty DocLinkSecret).
// Pass it to New() or copy and override specific fields as needed.
func testOptions() Options {
	return Options{
		DocLinkSecret: []byte("test-secret-for-unit-tests-only"),
	}
}
