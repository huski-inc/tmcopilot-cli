package tmc

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/huski-inc/tmcopilot-cli/internal/client"
)

var trademarkSerialWithInternalPrefix = regexp.MustCompile(`^[A-Z]{2}-TM-(.+)$`)

func normalizeCLIResponseData(data any) any {
	return normalizeResponseValue("", data)
}

func normalizeResponseValue(parentKey string, value any) any {
	switch typed := value.(type) {
	case *client.ResponseEnvelope:
		if typed == nil {
			return typed
		}
		return normalizeResponseEnvelope(*typed)
	case client.ResponseEnvelope:
		return normalizeResponseEnvelope(typed)
	case json.RawMessage:
		return normalizeRawJSONValue(typed)
	case []byte:
		return normalizeRawJSONValue(json.RawMessage(typed))
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, value := range typed {
			out[key] = normalizeResponseValue(key, value)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = normalizeResponseValue(parentKey, item)
		}
		return out
	case []map[string]any:
		out := make([]map[string]any, len(typed))
		for i, item := range typed {
			normalized, ok := normalizeResponseValue(parentKey, item).(map[string]any)
			if ok {
				out[i] = normalized
			} else {
				out[i] = item
			}
		}
		return out
	case string:
		if isTrademarkSerialField(parentKey) {
			return stripTrademarkSerialInternalPrefix(typed)
		}
		return typed
	default:
		return typed
	}
}

func normalizeResponseEnvelope(envelope client.ResponseEnvelope) map[string]any {
	out := map[string]any{
		"code":    envelope.Code,
		"message": envelope.Message,
	}
	if len(envelope.Data) > 0 {
		out["data"] = normalizeRawJSONValue(envelope.Data)
	}
	return out
}

func normalizeRawJSONValue(raw json.RawMessage) any {
	if len(raw) == 0 {
		return raw
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return raw
	}
	return normalizeResponseValue("", value)
}

func isTrademarkSerialField(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")
	if key == "sn" || key == "sns" || strings.HasSuffix(key, "_sn") || strings.HasSuffix(key, "_sns") {
		return true
	}
	return strings.Contains(key, "serial")
}

func stripTrademarkSerialInternalPrefix(value string) string {
	trimmed := strings.TrimSpace(value)
	matches := trademarkSerialWithInternalPrefix.FindStringSubmatch(trimmed)
	if len(matches) != 2 {
		return value
	}
	return matches[1]
}
