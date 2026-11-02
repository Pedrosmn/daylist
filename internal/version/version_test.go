package version

import "testing"

func withVersion(t *testing.T, version, commit, date string) {
	t.Helper()

	oldVersion, oldCommit, oldDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = oldVersion, oldCommit, oldDate
	})
	Version, Commit, Date = version, commit, date
}

func TestString(t *testing.T) {
	cases := []struct {
		name    string
		version string
		commit  string
		date    string
		want    string
	}{
		{"padrão", "dev", "unknown", "unknown", "dev"},
		{"commit e data", "1.2.3", "abc1234", "2026-11-02T10:00:00Z", "1.2.3 (commit abc1234, 2026-11-02T10:00:00Z)"},
		{"só commit", "1.2.3", "abc1234", "unknown", "1.2.3 (commit abc1234)"},
		{"só data", "1.2.3", "unknown", "2026-11-02T10:00:00Z", "1.2.3 (2026-11-02T10:00:00Z)"},
		{"metadados vazios", "1.2.3", "", "", "1.2.3"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withVersion(t, tc.version, tc.commit, tc.date)
			if got := String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}
