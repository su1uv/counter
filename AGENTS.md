# Counter

Counter is a Go service. A store manager asks, in Spanish or English, a question such as "¿Por qué bajó el margen en la tienda Centro esta semana?" An agent answers by calling typed SQL tools over sales, inventory, cash-register, and customer invoice data, and by retrieving passages from operating manuals and public SAT CFDI guides. Answers include citations.

## How work is done

- Do not assume. If a task is unclear, ask before starting.
- Every feature, bug, chore, or other activity that changes code starts with a plan. Split a complex activity into phases. One subagent writes each phase plan. Implementation starts only after Emilio approves that plan, and a different subagent implements it.
- Emilio specifies the model and the effort for every plan and every implementation. Do not choose them.
- Each plan or phase is a sequence of steps. Each step is one atomic, self-explanatory commit. Write simple, readable, maintainable code. Simplicity over ingenuity.
- Commit messages are short and use one of these prefixes: `feat:`, `chore:`, `bug:`, `misc:`, `docs:`.
- Use test-driven development. Aim for 100% coverage when that is meaningful. Prefer table-driven tests when they fit.
- When a task is finished, create a new local branch and commit the work there. Then stop. Ask Emilio before pushing, and before opening a draft pull request into `main`.
- Never push to `main`, and never push to a branch that runs an automated pipeline, such as staging or pre-prod.
- When a task finishes, reread this file. If it should change, show the proposed edit and wait for Emilio's explicit approval before changing it.
