JIRA CLI
========

A CLI for doing day-to-day JIRA operations via the terminal

Usage
-----

- `jira help [command]` - show help for jira or a subcommand
- `jira completion <shell>` - generate a shell autocompletion script
- `jira sprint [--project X] [--sort F,...]` - list my tickets in the current sprint, grouped by status; `--sort` as for `mine`
- `jira mine [--project X] [--open | --closed | --all | --status S ...] [--sort F,...]` - list my open tickets, grouped by status; `--open` (the default) for open only, `--closed` for closed only, `--all` for any status, `--status` (repeatable) for specific statuses. `--sort` takes any of `opened`, `updated`, `status`, `priority`, `type`, `project`, each optionally prefixed `+` (ascending) or `-` (descending); output is grouped by status only when `status` is first
- `jira watched [...]` - list open tickets I'm watching; same options and defaults as `mine`
- `jira watch <key|url>` - add me as a watcher of a ticket
- `jira unwatch <key|url>` - remove me as a watcher of a ticket
- `jira show <key|url>` - show a single ticket, with its description and comments as Markdown; fields from the project's `show_fields` in `~/.jira.yaml`
- `jira edit <key|url> [--dry-run]` - edit a ticket as Markdown in `$VISUAL`/`$EDITOR`: fields as YAML front matter (the project's `edit_fields`), description as the body
- `jira create [project] [--type T] [--dry-run]` - create a ticket the same way, from a template of the project's `create_fields`, `create_defaults` and required fields
- `jira comment <key|url>` - write a comment in `$VISUAL`/`$EDITOR` and post it; nothing is posted if the editor fails or the file is left empty
- `jira assign <key|url> [--user U] [--team T] [--dry-run]` - assign a ticket to you (the project's `me` fields), or to a user and/or team
- `jira delete <key|url> [--yes] [--subtasks]` - permanently delete a ticket, after confirmation
- `jira version` - print the version, commit and build date
- `jira whoami [--instance X | --project X]` - show who you are authenticated as on each instance in `~/.jira.yaml` (or just one), to validate the config
