// Package redact removes sensitive values (Binance request signatures and API
// keys) from strings before they are written to logs or returned to an MCP
// client. Binance signs requests by appending a `signature` HMAC to the URL
// query string; a leaked signed URL can be replayed within its recvWindow, so
// it must never reach a log file or the model/host consuming tool output.
//
// This addresses OWASP A09:2021 (Security Logging and Monitoring Failures) and
// the general principle of not exposing sensitive data (OWASP A02:2021).
package redact

import (
	"net/url"
	"regexp"
	"strings"
)

// placeholder replaces a redacted value.
const placeholder = "REDACTED"

// sensitiveParams are query-string keys whose values must never be exposed.
// Matched case-insensitively.
var sensitiveParams = map[string]struct{}{
	"signature": {},
	"apikey":    {},
	"token":     {},
	"secret":    {},
	"secretkey": {},
}

// sensitiveRe matches `key=value` pairs for sensitive keys inside an arbitrary
// string (e.g. an error message that embeds a signed URL). The value runs up to
// the next delimiter: `&`, whitespace, quotes, or end of string.
var sensitiveRe = regexp.MustCompile(`(?i)\b(signature|apikey|api_key|secret|secretkey|token)=[^&\s"']+`)

// URL returns rawURL with the values of any sensitive query parameters replaced
// by a placeholder. If rawURL cannot be parsed as a URL, it falls back to
// String so a malformed URL is never emitted verbatim.
func URL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return String(rawURL)
	}
	q := u.Query()
	changed := false
	for key := range q {
		if _, ok := sensitiveParams[strings.ToLower(key)]; ok {
			q.Set(key, placeholder)
			changed = true
		}
	}
	if !changed {
		return rawURL
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// String redacts sensitive `key=value` pairs found anywhere in s. Use it for
// free-form text such as error messages that may embed a signed URL.
func String(s string) string {
	return sensitiveRe.ReplaceAllStringFunc(s, func(match string) string {
		if i := strings.IndexByte(match, '='); i >= 0 {
			return match[:i+1] + placeholder
		}
		return match
	})
}

// Error redacts sensitive data from an error's message, returning "" for a nil
// error so callers can use it inline.
func Error(err error) string {
	if err == nil {
		return ""
	}
	return String(err.Error())
}
