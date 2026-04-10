package logging

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
)

const maxLoggedBodyLength = 8192

type ErrorLogger struct {
	file   *os.File
	logger *log.Logger
}

func NewErrorLogger(path string) (*ErrorLogger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	return &ErrorLogger{
		file:   file,
		logger: log.New(file, "", 0),
	}, nil
}

func (l *ErrorLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	return l.file.Close()
}

func (l *ErrorLogger) LogHTTPError(c *fiber.Ctx, status int, err error) {
	l.write("http_error", c, status, err, nil)
}

func (l *ErrorLogger) LogPanic(c *fiber.Ctx, recovered interface{}, stack []byte) {
	l.write("panic", c, fiber.StatusInternalServerError, recovered, stack)
}

func (l *ErrorLogger) write(kind string, c *fiber.Ctx, status int, err interface{}, stack []byte) {
	if l == nil || l.logger == nil || c == nil {
		return
	}

	payload := map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339Nano),
		"type":        kind,
		"status":      status,
		"method":      c.Method(),
		"path":        c.Path(),
		"originalURL": c.OriginalURL(),
		"ip":          c.IP(),
		"userAgent":   c.Get(fiber.HeaderUserAgent),
		"headers":     sanitizeHeaders(c.GetReqHeaders()),
		"query":       sanitizeMap(stringMap(c.Queries())),
		"params":      sanitizeMap(c.AllParams()),
		"body":        sanitizeBody(c),
	}

	if err != nil {
		payload["error"] = stringify(err)
	}

	if len(stack) > 0 {
		payload["stack"] = truncate(string(stack), maxLoggedBodyLength)
	}

	encoded, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		l.logger.Printf(`{"timestamp":"%s","type":"logger_error","error":"%s"}`, time.Now().Format(time.RFC3339Nano), truncate(marshalErr.Error(), 512))
		return
	}

	l.logger.Println(string(encoded))
}

func sanitizeHeaders(headers map[string][]string) map[string]interface{} {
	sanitized := make(map[string]interface{}, len(headers))
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if isSensitiveKey(lowerKey) {
			sanitized[key] = "***"
			continue
		}

		if len(values) == 1 {
			sanitized[key] = truncate(values[0], 512)
			continue
		}

		truncated := make([]string, 0, len(values))
		for _, value := range values {
			truncated = append(truncated, truncate(value, 512))
		}
		sanitized[key] = truncated
	}

	return sanitized
}

func sanitizeMap(values map[string]string) map[string]interface{} {
	sanitized := make(map[string]interface{}, len(values))
	for key, value := range values {
		if isSensitiveKey(strings.ToLower(key)) {
			sanitized[key] = "***"
			continue
		}
		sanitized[key] = truncate(value, 512)
	}
	return sanitized
}

func sanitizeBody(c *fiber.Ctx) interface{} {
	body := c.Body()
	if len(body) == 0 {
		return ""
	}

	contentType := strings.ToLower(c.Get(fiber.HeaderContentType))
	if strings.Contains(contentType, "multipart/form-data") {
		return "[multipart/form-data omitted]"
	}

	if json.Valid(body) {
		var parsed interface{}
		if err := json.Unmarshal(body, &parsed); err == nil {
			return sanitizeJSONValue(parsed)
		}
	}

	if !utf8.Valid(body) {
		return "[binary body omitted]"
	}

	return truncate(string(body), maxLoggedBodyLength)
}

func sanitizeJSONValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			if isSensitiveKey(strings.ToLower(key)) {
				sanitized[key] = "***"
				continue
			}
			sanitized[key] = sanitizeJSONValue(nested)
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, 0, len(typed))
		for _, nested := range typed {
			sanitized = append(sanitized, sanitizeJSONValue(nested))
		}
		return sanitized
	case string:
		return truncate(typed, maxLoggedBodyLength)
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	sensitiveWords := []string{"authorization", "cookie", "token", "secret", "password", "session", "jwt"}
	for _, word := range sensitiveWords {
		if strings.Contains(key, word) {
			return true
		}
	}
	return false
}

func stringify(value interface{}) string {
	switch typed := value.(type) {
	case error:
		return truncate(typed.Error(), maxLoggedBodyLength)
	case string:
		return truncate(typed, maxLoggedBodyLength)
	default:
		return truncate(strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(toJSON(value)), "\n", " "), "\t", " ")), maxLoggedBodyLength)
	}
}

func toJSON(value interface{}) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "unserializable error"
	}
	return string(encoded)
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "...(truncated)"
}

func stringMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	return values
}
