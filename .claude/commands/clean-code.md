---
description: Perform a Clean Code and idiomatic Go refactor review.
---
Review the provided Go files for readability and architectural soundness.

Criteria:
1. **SOLID & DRY:** Identify redundant logic and violated responsibilities.
2. **Design Patterns:** Look for opportunities to apply design patterns (GoF) to improve modularity and separation of concerns.
3. **Naming:** Enforce descriptive names. Replace cryptic abbreviations (e.g., 'm', 'msg', 'cmd') with clear terminology, unless they are standard Go idioms (like 'r' for receiver).
4. **Small Functions:** Flag functions exceeding 40-50 lines.
5. **TUI Structure:** Ensure clear separation of state (Model), logic (Update), and presentation (View).

Special Considerations:
1. If a file or a function is very performance-critical, you may allow some leniency in terms of readability and modularity, but only if it is justified with a comment explaining the performance considerations.

Output Format:
- **Summary:** A brief list of identified "code smells".
- **Refactoring:** Provide the refactored code blocks with a brief "Rationale" for each change.

Target: $ARGUMENTS
