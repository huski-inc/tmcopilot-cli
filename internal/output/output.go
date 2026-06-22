package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Success struct {
	OK   bool           `json:"ok"`
	Data any            `json:"data,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
}

type Failure struct {
	OK          bool     `json:"ok"`
	Type        string   `json:"type"`
	Message     string   `json:"message"`
	Hint        string   `json:"hint,omitempty"`
	StatusCode  int      `json:"status_code,omitempty"`
	Code        int      `json:"code,omitempty"`
	TraceID     string   `json:"trace_id,omitempty"`
	Retryable   bool     `json:"retryable,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func Write(format, outputPath string, data any, meta map[string]any) error {
	return WriteTo(os.Stdout, format, outputPath, data, meta)
}

func WriteTo(w io.Writer, format, outputPath string, data any, meta map[string]any) error {
	if w == nil {
		w = os.Stdout
	}
	if format == "" {
		format = "json"
	}
	switch format {
	case "json", "pretty":
		payload := Success{OK: true, Data: data, Meta: meta}
		raw, err := marshal(format, payload)
		if err != nil {
			return err
		}
		if outputPath != "" {
			return writeFile(outputPath, raw)
		}
		_, err = w.Write(raw)
		return err
	case "raw":
		raw, err := marshal("json", data)
		if err != nil {
			return err
		}
		if outputPath != "" {
			return writeFile(outputPath, raw)
		}
		_, err = w.Write(raw)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func WriteError(w io.Writer, message string, traceID string) {
	WriteFailure(w, Failure{
		OK:      false,
		Type:    "error",
		Message: message,
		TraceID: traceID,
	})
}

func WriteFailure(w io.Writer, payload Failure) {
	if w == nil {
		w = os.Stderr
	}
	payload.OK = false
	if payload.Type == "" {
		payload.Type = "error"
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(w, payload.Message)
		return
	}
	fmt.Fprintln(w, string(raw))
}

func marshal(format string, value any) ([]byte, error) {
	var (
		raw []byte
		err error
	)
	if format == "pretty" {
		raw, err = json.MarshalIndent(value, "", "  ")
	} else {
		raw, err = json.Marshal(value)
	}
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}

func writeFile(path string, raw []byte) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func CreateFile(path string) (*os.File, error) {
	if err := ensureParentDir(path); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, err
	}
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func WriteRawFile(path string, raw []byte) error {
	return writeFile(path, raw)
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
