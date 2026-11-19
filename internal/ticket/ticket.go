// Package ticket finds issue-tracker IDs (e.g. AXE-123) and renders them
// into commit message templates.
package ticket

import (
	"fmt"
	"regexp"
	"strings"
)

// DefaultPattern matches Jira-style keys such as ONE-0001 or AXE-123.
const DefaultPattern = `[A-Z][A-Z0-9]+-\d+`

// DefaultTemplate is used when a ticket is found and aicm.format is unset.
const DefaultTemplate = "{ticket}: {message}"

// FromBranch returns the first match of pattern in branch, or "" if none.
func FromBranch(branch, pattern string) (string, error) {
	if pattern == "" {
		pattern = DefaultPattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid aicm.ticketPattern %q: %w", pattern, err)
	}
	return re.FindString(branch), nil
}

// Resolve picks the template to apply for ticket, if any. A template that
// mentions {ticket} applies only when there is a ticket; one that doesn't
// applies to every message. With no template, DefaultTemplate applies when
// there is a ticket. applies is false when the message should be used as is.
func Resolve(template, ticket string) (tmpl string, applies bool, err error) {
	switch {
	case template == "":
		if ticket == "" {
			return "", false, nil
		}
		return DefaultTemplate, true, nil
	case !strings.Contains(template, "{message}"):
		return "", false, fmt.Errorf("aicm.format %q must contain {message}", template)
	case strings.Contains(template, "{ticket}") && ticket == "":
		return "", false, nil
	}
	return template, true, nil
}

// Apply renders the template onto the subject line of message. The body
// (everything after the first line) is kept unchanged.
func Apply(template, ticket, message string) string {
	if template == "" {
		template = DefaultTemplate
	}
	subject, body, _ := strings.Cut(message, "\n")
	subject = strings.NewReplacer("{ticket}", ticket, "{message}", subject).Replace(template)
	if body == "" {
		return subject
	}
	return subject + "\n" + body
}
