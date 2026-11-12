package dotenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := strings.Join([]string{
		"# comment",
		"TG_BOT_TOKEN=123:abc",
		`export TG_CHAT_ID="@my_channel"`,
		"QUOTED='a # b'",
		"INLINE=0.12 # threshold",
		"ALREADY=from-file",
	}, "\n") + "\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, k := range []string{"TG_BOT_TOKEN", "TG_CHAT_ID", "QUOTED", "INLINE"} {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}

	t.Setenv("ALREADY", "from-env")

	applied, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"TG_BOT_TOKEN": "123:abc",
		"TG_CHAT_ID":   "@my_channel",
		"QUOTED":       "a # b",
		"INLINE":       "0.12",
		"ALREADY":      "from-env",
	}

	for k, v := range want {
		if got := os.Getenv(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}

	if applied != 4 {
		t.Errorf("applied %d, want 4", applied)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if applied, err := Load(filepath.Join(t.TempDir(), "missing")); applied != 0 || err != nil {
		t.Fatalf("got (%d, %v)", applied, err)
	}
}

func TestLoadRejectsMalformedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")

	if err := os.WriteFile(path, []byte("NOT_A_PAIR\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("malformed line accepted")
	}
}
