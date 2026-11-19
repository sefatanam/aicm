# Glossary

**Commit Message**
The text recorded with a git commit: a one-line subject, optionally followed by a blank line and a body.

**Staged Changes**
The changes added to the git index and waiting to be committed. They are the only input aicm writes a message from.

**Ticket**
An issue-tracker ID such as `AXE-123` or `ONE-0001` that links a commit to a piece of work. Usually taken from the branch name.

**Template**
The pattern that shapes the final subject line around the generated message, e.g. `{ticket}: {message}` or `[web] {message}`. A Template that mentions the Ticket is only used when there is a Ticket; one that doesn't is used for every Commit Message. With a Ticket and no Template configured, `{ticket}: {message}` is used.

**Style Prompt**
The instructions that define how a message is written (by default, Conventional Commits). The user may replace it entirely; what aicm needs to work (output only the message, leave Tickets out, the Variant format) is never part of it and always applies.

**Rules**
Free-text style instructions from the user (e.g. "lowercase, no period") added on top of the Style Prompt, taking priority over it.

**Provider**
The AI service that writes the Commit Message. Claude is the default; Ollama, Codex and Gemini are planned.

**Variant**
One of the candidate Commit Messages offered for the same Staged Changes. When there is a Ticket, aicm offers two: one with a Conventional Commits type and one without.

**Conventional Commits**
The default Style Prompt: the message style used when nothing else is configured: `<type>(<scope>): <subject>`, with types such as `feat`, `fix`, `chore` and `refactor`.
