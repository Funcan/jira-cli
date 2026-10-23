package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// errNoChanges is returned when the user exits the editor without changing
// the file, or the changes don't change any field.
var errNoChanges = errors.New("no changes made, aborting")

// editLoop writes text to a temporary Markdown file and opens it in the
// user's editor, then passes the edited text to apply. If apply fails, the
// error is shown and the user can edit the file again. The file is removed
// unless the user gives up after an error, in which case its path is printed
// so the work isn't lost.
func editLoop(cmd *cobra.Command, text string, apply func(text string) error) error {
	f, err := os.CreateTemp("", "jira-*.md")
	if err != nil {
		return err
	}
	path := f.Name()
	_, err = f.WriteString(text)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(path)
		return err
	}

	stdin := bufio.NewReader(cmd.InOrStdin())
	for {
		if err := runEditor(cmd, path); err != nil {
			_ = os.Remove(path)
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		edited := string(data)
		if edited == text {
			_ = os.Remove(path)
			return errNoChanges
		}
		text = edited

		err = apply(text)
		if err == nil || errors.Is(err, errNoChanges) {
			_ = os.Remove(path)
			return err
		}
		fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		fmt.Fprint(cmd.ErrOrStderr(), "Edit again? [Y/n] ")
		answer, readErr := stdin.ReadString('\n')
		answer = strings.ToLower(strings.TrimSpace(answer))
		if readErr != nil && answer == "" {
			// No answer (e.g. stdin isn't a terminal) means no.
			answer = "n"
			fmt.Fprintln(cmd.ErrOrStderr())
		}
		if answer != "" && answer != "y" && answer != "yes" {
			return fmt.Errorf("aborted; your changes are in %s", path)
		}
	}
}

// runEditor opens path in $VISUAL, $EDITOR or vi. Like git, the editor is
// run by the shell so it can include arguments.
func runEditor(cmd *cobra.Command, path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	c := exec.CommandContext(cmd.Context(), "sh", "-c", editor+` "$@"`, editor, path)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("running editor %q: %w", editor, err)
	}
	return nil
}
