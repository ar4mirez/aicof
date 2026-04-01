package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON_Success(t *testing.T) {
	output := captureOutput(t, func() {
		PrintJSON("test-cmd", map[string]string{"key": "value"})
	})

	var resp JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, output)
	}

	if resp.Command != "test-cmd" {
		t.Errorf("expected command=test-cmd, got %s", resp.Command)
	}
	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Error != "" {
		t.Errorf("expected no error, got %s", resp.Error)
	}
	if resp.Data == nil {
		t.Error("expected data to be non-nil")
	}
}

func TestPrintJSON_NilData(t *testing.T) {
	output := captureOutput(t, func() {
		PrintJSON("empty", nil)
	})

	var resp JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}
	if resp.Data != nil {
		t.Error("expected data to be nil (omitted)")
	}
}

func TestPrintJSONError(t *testing.T) {
	output := captureStderr(t, func() {
		PrintJSONError("fail-cmd", errors.New("something broke"))
	})

	var resp JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, output)
	}

	if resp.Command != "fail-cmd" {
		t.Errorf("expected command=fail-cmd, got %s", resp.Command)
	}
	if resp.Success {
		t.Error("expected success=false")
	}
	if resp.Error != "something broke" {
		t.Errorf("expected error='something broke', got %s", resp.Error)
	}
}

func TestPrintJSON_ComplexData(t *testing.T) {
	data := map[string]interface{}{
		"count": 42,
		"items": []string{"a", "b", "c"},
		"nested": map[string]interface{}{
			"deep": true,
		},
	}

	output := captureOutput(t, func() {
		PrintJSON("complex", data)
	})

	var resp JSONResponse
	if err := json.Unmarshal([]byte(output), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if !resp.Success {
		t.Error("expected success=true")
	}

	m := resp.Data.(map[string]interface{})
	if m["count"].(float64) != 42 {
		t.Errorf("expected count=42, got %v", m["count"])
	}
}
