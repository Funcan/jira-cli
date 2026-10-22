JIRA CLI
========

A CLI for doing day-to-day JIRA operations via the terminal

Usage
-----

- `jira help [command]` - show help for jira or a subcommand
- `jira completion <shell>` - generate a shell autocompletion script
- `jira sprint [--project X] [--sort F,...]` - list my tickets in the current sprint, grouped by status; `--sort` as for `mine`
- `jira mine [--project X] [--open | --closed | --all | --status S ...] [--sort F,...]` - list my open tickets, grouped by status; `--open` (the default) for open only, `--closed` for closed only, `--all` for any status, `--status` (repeatable) for specific statuses. `--sort` takes any of `opened`, `updated`, `status`, `priority`, `type`, `project`, each optionally prefixed `+` (ascending) or `-` (descending); output is grouped by status only when `status` is first
- `jira version` - print the version, commit and build date
- `jira whoami [--instance X | --project X]` - show who you are authenticated as on each instance in `~/.jira.yaml` (or just one), to validate the config
