package runplan

import (
	"bufio"
	"fmt"
	"regexp"
	"strings"
)

// LogSeverity represents the severity level of a classified log event.
type LogSeverity string

const (
	LogSeverityFatal    LogSeverity = "fatal"
	LogSeverityError    LogSeverity = "error"
	LogSeverityWarning  LogSeverity = "warning"
	LogSeverityAdvisory LogSeverity = "advisory"
	LogSeverityNoise    LogSeverity = "noise"
)

// LogCategory represents the category of a classified log event.
type LogCategory string

const (
	LogCategoryDependencyWarning LogCategory = "dependency_warning"
	LogCategoryDefaultSelection  LogCategory = "default_selection"
	LogCategoryArgConflict       LogCategory = "arg_conflict"
	LogCategoryOOM               LogCategory = "oom"
	LogCategoryHealth            LogCategory = "health"
	LogCategoryStartup           LogCategory = "startup"
)

// RuntimeLogRule defines a pattern to match against runtime log lines.
type RuntimeLogRule struct {
	ID         string
	Backend    string // vllm, sglang, llamacpp, ollama, * (any)
	Version    string // optional
	Pattern    *regexp.Regexp
	Severity   LogSeverity
	Category   LogCategory
	Message    string
	Suggestion string
}

// RuntimeLogEvent represents a classified log event.
type RuntimeLogEvent struct {
	RuleID      string      `json:"rule_id"`
	Severity    LogSeverity `json:"severity"`
	Category    LogCategory `json:"category"`
	Message     string      `json:"message"`
	Suggestion  string      `json:"suggestion"`
	RawLine     string      `json:"raw_line"`
	Occurrences int         `json:"occurrences"`
}

// RuntimeLogClassifier classifies log lines against a set of rules.
type RuntimeLogClassifier struct {
	rules []RuntimeLogRule
}

// NewRuntimeLogClassifier creates a classifier with the default built-in rules.
func NewRuntimeLogClassifier() *RuntimeLogClassifier {
	c := &RuntimeLogClassifier{}
	c.addBuiltinRules()
	return c
}

// ClassifyLogText classifies all lines in the given log text.
// Returns events grouped by rule ID with occurrence counts.
func (c *RuntimeLogClassifier) ClassifyLogText(logText string) []RuntimeLogEvent {
	counts := make(map[string]*RuntimeLogEvent)
	scanner := bufio.NewScanner(strings.NewReader(logText))
	for scanner.Scan() {
		line := scanner.Text()
		for _, rule := range c.rules {
			if rule.Pattern.MatchString(line) {
				if existing, ok := counts[rule.ID]; ok {
					existing.Occurrences++
				} else {
					counts[rule.ID] = &RuntimeLogEvent{
						RuleID:      rule.ID,
						Severity:    rule.Severity,
						Category:    rule.Category,
						Message:     rule.Message,
						Suggestion:  rule.Suggestion,
						RawLine:     line,
						Occurrences: 1,
					}
				}
				break // first matching rule wins per line
			}
		}
	}
	events := make([]RuntimeLogEvent, 0, len(counts))
	for _, ev := range counts {
		events = append(events, *ev)
	}
	return events
}

// ClassifyLogLines classifies a slice of log lines.
func (c *RuntimeLogClassifier) ClassifyLogLines(lines []string) []RuntimeLogEvent {
	return c.ClassifyLogText(strings.Join(lines, "\n"))
}

// addBuiltinRules registers the initial set of known log patterns.
func (c *RuntimeLogClassifier) addBuiltinRules() {
	c.rules = []RuntimeLogRule{
		{
			ID:         "sglang.torchao.syntax_warning",
			Backend:    "sglang",
			Pattern:    regexp.MustCompile(`torchao/quantization/.*SyntaxWarning: invalid escape sequence`),
			Severity:   LogSeverityNoise,
			Category:   LogCategoryDependencyWarning,
			Message:    "Upstream dependency torchao emits a SyntaxWarning. This does not affect model serving.",
			Suggestion: "No action needed. This is a known upstream issue in torchao.",
		},
		{
			ID:         "sglang.attention_backend.default",
			Backend:    "sglang",
			Pattern:    regexp.MustCompile(`Attention backend not specified.*Use.*backend by default`),
			Severity:   LogSeverityAdvisory,
			Category:   LogCategoryDefaultSelection,
			Message:    "SGLang used the default attention backend (flashinfer).",
			Suggestion: "If flashinfer-related failures occur, set --attention-backend to another supported value.",
		},
		{
			ID:         "llamacpp.env_overwritten.host",
			Backend:    "llamacpp",
			Pattern:    regexp.MustCompile(`LLAMA_ARG_HOST.*environment variable is set.*overwritten.*--host`),
			Severity:   LogSeverityWarning,
			Category:   LogCategoryArgConflict,
			Message:    "LLAMA_ARG_HOST environment variable is set but will be overwritten by --host argument.",
			Suggestion: "This is typically caused by the Docker image providing LLAMA_ARG_HOST as a default env. The platform's --host correctly overrides it. If you set LLAMA_ARG_HOST yourself, remove it to avoid this warning.",
		},
		{
			ID:         "llamacpp.env_overwritten.port",
			Backend:    "llamacpp",
			Pattern:    regexp.MustCompile(`LLAMA_ARG_PORT.*environment variable is set.*overwritten.*--port`),
			Severity:   LogSeverityWarning,
			Category:   LogCategoryArgConflict,
			Message:    "LLAMA_ARG_PORT environment variable is set but will be overwritten by --port argument.",
			Suggestion: "This is typically caused by the Docker image providing LLAMA_ARG_PORT as a default env. The platform's --port correctly overrides it. If you set LLAMA_ARG_PORT yourself, remove it to avoid this warning.",
		},
		{
			ID:         "cuda.oom",
			Backend:    "*",
			Pattern:    regexp.MustCompile(`(?i)(CUDA out of memory|out of memory|RuntimeError.*out of memory)`),
			Severity:   LogSeverityError,
			Category:   LogCategoryOOM,
			Message:    "CUDA out of memory error detected.",
			Suggestion: "Reduce model size, context length, batch size, or memory budget. Consider using a smaller model or fewer GPU layers.",
		},
		{
			ID:         "container.startup.failed",
			Backend:    "*",
			Pattern:    regexp.MustCompile(`(?i)(Traceback \(most recent call last\)|panic:|fatal error|failed to start|Error.*failed to load)`),
			Severity:   LogSeverityError,
			Category:   LogCategoryStartup,
			Message:    "Container startup failure detected.",
			Suggestion: "Check the full container logs for details. Common causes: missing model files, incompatible GPU drivers, port conflicts.",
		},
	}
}

// RegisterRule adds a custom rule to the classifier.
func (c *RuntimeLogClassifier) RegisterRule(rule RuntimeLogRule) {
	c.rules = append(c.rules, rule)
}

// FilterBySeverity returns only events with severity at or above the given threshold.
func FilterBySeverity(events []RuntimeLogEvent, minSeverity LogSeverity) []RuntimeLogEvent {
	severityOrder := map[LogSeverity]int{
		LogSeverityNoise:    0,
		LogSeverityAdvisory: 1,
		LogSeverityWarning:  2,
		LogSeverityError:    3,
		LogSeverityFatal:    4,
	}
	minLevel, ok := severityOrder[minSeverity]
	if !ok {
		return events
	}
	var filtered []RuntimeLogEvent
	for _, ev := range events {
		if level, ok := severityOrder[ev.Severity]; ok && level >= minLevel {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}

// IsNonFatal returns true if all events are noise or advisory (should not change instance state).
func IsNonFatal(events []RuntimeLogEvent) bool {
	for _, ev := range events {
		if ev.Severity != LogSeverityNoise && ev.Severity != LogSeverityAdvisory {
			return false
		}
	}
	return true
}

// FormatEventsForDisplay formats log events for user-facing display.
func FormatEventsForDisplay(events []RuntimeLogEvent) string {
	if len(events) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, ev := range events {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("[%s] %s: %s", ev.Severity, ev.RuleID, ev.Message))
		if ev.Suggestion != "" {
			sb.WriteString(fmt.Sprintf(" — %s", ev.Suggestion))
		}
		if ev.Occurrences > 1 {
			sb.WriteString(fmt.Sprintf(" (x%d)", ev.Occurrences))
		}
	}
	return sb.String()
}
