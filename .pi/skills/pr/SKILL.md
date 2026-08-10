---
name: pr
description: Create or manage a GitHub pull request following this repository's conventions. Use when the user asks to create, open, raise, update, or respond to a PR.
---

# Create Draft PR

Create a draft pull request for the current branch using the `gh` CLI. Follow every rule in this skill exactly.

## Tone

The PR title and body should be friendly, constructive, and professional.

- Frame changes positively.
- Avoid accusatory or blaming language about prior code or authors.
- Prefer collaborative phrasing such as "we" and "this PR".
- Avoid sarcasm, dismissive shorthand, and hype.
- Describe trade-offs or limitations as useful reviewer context.

## Pre-flight checks

1. Confirm the current directory is inside a Git repository and the current branch is not the default branch. If it is the default branch, stop and ask which branch to use.
2. Run `git status` and check for uncommitted changes that should be included. If any exist, stop and ask the user before proceeding.
3. Determine the default branch from the remote, then run `git log <default-branch>..HEAD --oneline`. If the branch is not ahead of the default branch, stop and report this.
4. Inspect `git diff <default-branch>...HEAD` to understand the complete PR rather than relying only on commit messages.
5. Push unpushed commits with `git push`. If no upstream is configured, use `git push -u origin <branch>`. Never force push. If a force push appears necessary, stop and ask the user to do it.

## Locate the PR template

Use `.github/pull_request_template.md` as the basis for the PR body. If it is missing, stop and ask the user whether to proceed without it. Do not invent a replacement template.

## Compose the PR title

- Use the Conventional Commits format: `<type>[optional scope]: <description>`.
- Write a short, imperative description derived from the complete branch diff and commits.
- Choose the type that best represents the complete PR, such as `feat`, `fix`, `chore`, `docs`, `refactor`, `test`, `build`, or `ci`.
- Do not add a work-item identifier or any prefix before the conventional commit type unless the user explicitly requests one.
- Do not copy a long commit message verbatim.
- Keep the title under approximately 70 characters where possible.

## Compose the PR body

Fill the repository's PR template with one brief paragraph summarising the changes made in the PR.

- Copy `.github/pull_request_template.md` to a temporary Markdown file.
- Replace its summary prompt with the completed paragraph so no placeholder remains in the submitted body.
- Keep the summary to a single paragraph with no headings, bullet points, or checklist.
- Describe the outcome and the key implementation approach when useful.
- Focus on the complete PR diff against the default branch.
- Do not invent behaviour, motivation, or testing results.
- Preserve any other template content if the template is expanded in the future.

## Create the PR

Run `gh pr create` with:

- `--draft` so the user can decide when the PR is ready for review.
- `--title "<title>"` using the composed title.
- `--body-file <temporary-markdown-file>` using the completed PR template.
- `--base <default-branch>` when the base branch is not detected correctly.

Example:

```bash
gh pr create \
  --draft \
  --title "feat: add workspace search filters" \
  --body-file "$pr_body_file"
```

Do not:

- Submit the PR as ready for review or run `gh pr ready`.
- Use `--fill` in place of the explicit title and body.
- Force push under any circumstances.
- Add a changelog entry unless the user explicitly asks for one.

## After creation

1. Remove the temporary body file.
2. Print the PR URL returned by `gh pr create`.
3. Remind the user that the PR is a draft and they can mark it ready when appropriate.

## Managing an existing PR

Use the `gh` CLI for all interactions with an existing PR. Preserve the concise, constructive tone when updating its title or body or replying to review comments. Sign replies with the current agent name.
