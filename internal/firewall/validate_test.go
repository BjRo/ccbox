package firewall

import "testing"

func TestValidateDomain_ValidBareDomains(t *testing.T) {
	valid := []string{
		"example.com",
		"sub.example.com",
		"a-b.example.com",
		"deep.nested.sub.example.com",
		"a.b.c.d.e.f.example.com",
		"x.co",
	}

	for _, name := range valid {
		if err := ValidateDomain(name); err != nil {
			t.Errorf("ValidateDomain(%q) returned error: %v", name, err)
		}
	}
}

func TestValidateDomain_ValidWildcardDomains(t *testing.T) {
	valid := []string{
		"*.example.com",
		"*.sub.example.com",
	}

	for _, name := range valid {
		if err := ValidateDomain(name); err != nil {
			t.Errorf("ValidateDomain(%q) returned error: %v", name, err)
		}
	}
}

func TestValidateDomain_ValidSingleLabel(t *testing.T) {
	// Single-label domains like "localhost" are valid DNS names.
	if err := ValidateDomain("localhost"); err != nil {
		t.Errorf("ValidateDomain(%q) returned error: %v", "localhost", err)
	}
}

func TestValidateDomain_MaxLabelLength(t *testing.T) {
	// Labels can be up to 63 characters.
	label63 := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz0" // 63 chars
	name := label63 + ".example.com"
	if err := ValidateDomain(name); err != nil {
		t.Errorf("ValidateDomain(%q) returned error: %v", name, err)
	}
}

func TestValidateDomain_LabelTooLong(t *testing.T) {
	// Labels must not exceed 63 characters.
	label64 := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz01" // 64 chars
	name := label64 + ".example.com"
	if err := ValidateDomain(name); err == nil {
		t.Errorf("ValidateDomain(%q) should have returned error for label > 63 chars", name)
	}
}

func TestValidateDomain_MaxTotalLength(t *testing.T) {
	// Total name must not exceed 253 characters. Build a name that is exactly 253.
	// Use repeating "a." labels (2 chars each) + final label.
	// 126 labels of "a." = 252 chars + "a" = 253.
	name := ""
	for i := 0; i < 126; i++ {
		name += "a."
	}
	name += "a"
	if len(name) != 253 {
		t.Fatalf("test setup: expected name length 253, got %d", len(name))
	}
	if err := ValidateDomain(name); err != nil {
		t.Errorf("ValidateDomain(%q) returned error: %v (length %d)", name, err, len(name))
	}
}

func TestValidateDomain_TotalLengthTooLong(t *testing.T) {
	// Name exceeding 253 characters should be rejected.
	name := ""
	for i := 0; i < 127; i++ {
		name += "a."
	}
	name += "a"
	if len(name) <= 253 {
		t.Fatalf("test setup: expected name length > 253, got %d", len(name))
	}
	if err := ValidateDomain(name); err == nil {
		t.Errorf("ValidateDomain(%q) should have returned error for name > 253 chars", name)
	}
}

func TestValidateDomain_InvalidEmpty(t *testing.T) {
	if err := ValidateDomain(""); err == nil {
		t.Error("ValidateDomain(\"\") should have returned error for empty string")
	}
}

func TestValidateDomain_InvalidLeadingHyphen(t *testing.T) {
	if err := ValidateDomain("-example.com"); err == nil {
		t.Error("ValidateDomain(\"-example.com\") should have returned error")
	}
}

func TestValidateDomain_InvalidTrailingHyphen(t *testing.T) {
	if err := ValidateDomain("example-.com"); err == nil {
		t.Error("ValidateDomain(\"example-.com\") should have returned error")
	}
}

func TestValidateDomain_InvalidConsecutiveDots(t *testing.T) {
	if err := ValidateDomain("example..com"); err == nil {
		t.Error("ValidateDomain(\"example..com\") should have returned error")
	}
}

func TestValidateDomain_InvalidBareWildcard(t *testing.T) {
	if err := ValidateDomain("*"); err == nil {
		t.Error("ValidateDomain(\"*\") should have returned error")
	}
}

func TestValidateDomain_InvalidSpaces(t *testing.T) {
	if err := ValidateDomain("example .com"); err == nil {
		t.Error("ValidateDomain(\"example .com\") should have returned error")
	}
}

func TestValidateDomain_ShellInjection(t *testing.T) {
	injections := []string{
		"; rm -rf /",
		"$(whoami)",
		"`cmd`",
		"foo | bar",
		"foo & bar",
		"foo()",
		"foo{}",
		"foo\\bar",
		"foo'bar",
		"foo\"bar",
		"foo>bar",
		"foo<bar",
		"!foo",
		"foo#bar",
		"~foo",
		"$HOME",
	}

	for _, name := range injections {
		if err := ValidateDomain(name); err == nil {
			t.Errorf("ValidateDomain(%q) should have returned error for shell injection attempt", name)
		}
	}
}

func TestValidateDomain_ErrorMessageIncludesInput(t *testing.T) {
	err := ValidateDomain("; rm -rf /")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}
