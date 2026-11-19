# Glossary

**Commit Message**
The text recorded with a git commit: a one-line subject, optionally followed by a blank line and a body.

**Staged Changes**
The changes added to the git index and waiting to be committed. They are the only input aicm writes a message from.

**Ticket**
An issue-tracker ID such as `AXE-123` or `ONE-0001` that links a commit to a piece of work. Usually taken from the branch name.

**Template**
The pattern that places the Ticket and the generated message into the final subject line, e.g. `{ticket}: {message}`. Only used when there is a Ticket.

**Rules**
Free-text style instructions from the user (e.g. "lowercase, no period") that the Provider follows when writing the message.

**Provider**
The AI service that writes the Commit Message. Claude is the default; Ollama, Codex and Gemini are planned.

**Variant**
One of the candidate Commit Messages offered for the same Staged Changes. When there is a Ticket, aicm offers two: one with a Conventional Commits type and one without.

**Conventional Commits**
The default message style when nothing else is configured: `<type>(<scope>): <subject>`, with types such as `feat`, `fix`, `chore` and `refactor`.
