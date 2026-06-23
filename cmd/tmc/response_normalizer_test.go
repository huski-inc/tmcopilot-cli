package tmc

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/huski-inc/tmcopilot-cli/internal/client"
)

func TestNormalizeCLIResponseDataStripsTrademarkSerialPrefixes(t *testing.T) {
	raw := json.RawMessage(`{
		"items": [
			{
				"id": "US-TM-99999999",
				"serial_number": "US-TM-88418692",
				"serial_number_id": "US-TM-88418693",
				"tm_serial_number": "CA-TM-88418694",
				"owner_id": "US-TM-88888888"
			}
		],
		"serial_numbers": ["US-TM-11111111", "88418692"],
		"nested": {
			"case_sn": "US-TM-22222222",
			"height_risk_sns": [1, 2]
		}
	}`)

	normalized := normalizeCLIResponseData(raw)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		t.Fatalf("marshal normalized data: %v", err)
	}
	output := string(encoded)
	for _, forbidden := range []string{
		`"serial_number":"US-TM-88418692"`,
		`"serial_number_id":"US-TM-88418693"`,
		`"tm_serial_number":"CA-TM-88418694"`,
		`"US-TM-11111111"`,
		`"case_sn":"US-TM-22222222"`,
	} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("normalized output still contains %s: %s", forbidden, output)
		}
	}
	for _, want := range []string{
		`"serial_number":"88418692"`,
		`"serial_number_id":"88418693"`,
		`"tm_serial_number":"88418694"`,
		`"11111111"`,
		`"case_sn":"22222222"`,
		`"id":"US-TM-99999999"`,
		`"owner_id":"US-TM-88888888"`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("normalized output missing %s: %s", want, output)
		}
	}
}

func TestWriteResultNormalizesTrademarkSerialPrefixes(t *testing.T) {
	var out bytes.Buffer
	rt := &runtimeContext{
		Format: "json",
		Out:    &out,
	}
	raw := json.RawMessage(`{"serial_number":"US-TM-88418692","id":"US-TM-99999999"}`)

	if err := writeResult(rt, raw, nil); err != nil {
		t.Fatalf("writeResult failed: %v", err)
	}
	output := out.String()
	if strings.Contains(output, `"serial_number":"US-TM-88418692"`) {
		t.Fatalf("serial prefix was not stripped: %s", output)
	}
	if !strings.Contains(output, `"serial_number":"88418692"`) {
		t.Fatalf("serial number missing normalized value: %s", output)
	}
	if !strings.Contains(output, `"id":"US-TM-99999999"`) {
		t.Fatalf("non-serial id should be preserved: %s", output)
	}
}

func TestWriteResultNormalizesRawEnvelopeTrademarkSerialPrefixes(t *testing.T) {
	var out bytes.Buffer
	rt := &runtimeContext{
		Format: "json",
		Out:    &out,
	}
	envelope := &client.ResponseEnvelope{
		Code:    0,
		Message: client.Message{Title: "OK", Text: "ok"},
		Data:    json.RawMessage(`{"serial_number":"US-TM-88418692","id":"US-TM-99999999"}`),
	}

	if err := writeResult(rt, envelope, nil); err != nil {
		t.Fatalf("writeResult failed: %v", err)
	}
	output := out.String()
	if strings.Contains(output, `"serial_number":"US-TM-88418692"`) {
		t.Fatalf("raw envelope serial prefix was not stripped: %s", output)
	}
	if !strings.Contains(output, `"serial_number":"88418692"`) {
		t.Fatalf("raw envelope serial number missing normalized value: %s", output)
	}
	if !strings.Contains(output, `"id":"US-TM-99999999"`) {
		t.Fatalf("raw envelope non-serial id should be preserved: %s", output)
	}
}
