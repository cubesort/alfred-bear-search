package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/drgrib/alfred"
)

func TestErrorOutput(t *testing.T) {
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	diagnostic, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer diagnostic.Close()
	oldOut, oldErr := os.Stdout, os.Stderr
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	os.Stdout, os.Stderr = out, diagnostic
	reportError(errors.New("test database unavailable"))
	alfred.Run()
	os.Stdout, os.Stderr = oldOut, oldErr
	out.Seek(0, 0)
	diagnostic.Seek(0, 0)
	var result struct {
		Items []struct {
			Title string
			Valid bool
		}
	}
	if err = json.NewDecoder(out).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Valid || result.Items[0].Title != "Unable to search Bear notes" {
		t.Fatalf("unexpected error result: %+v", result)
	}
	message, _ := io.ReadAll(diagnostic)
	if !strings.Contains(string(message), "test database unavailable") {
		t.Fatal("missing diagnostic")
	}
}
