package ghactions

import (
	"os"
	"testing"
)

func TestUnmarshalInputs(t *testing.T) {
	os.Setenv("GITHUB_EVENT_PATH", "testdata/event.json")
	defer os.Unsetenv("GITHUB_EVENT_PATH")

	var inputs struct {
		Foo string `json:"foo"`
	}

	if err := UnmarshalInputs(&inputs); err != nil {
		t.Fatal(err)
	}

	if inputs.Foo != "bar" {
		t.Fatalf("want bar, got %s", inputs.Foo)
	}
}
