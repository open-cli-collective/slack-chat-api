// Package watch implements `slck watch`: follow messages for replies and
// reactions as Slack pushes them, instead of polling.
package watch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/slack-chat-api/internal/client"
	"github.com/open-cli-collective/slack-chat-api/internal/messageref"
	"github.com/open-cli-collective/slack-chat-api/internal/output"
	"github.com/open-cli-collective/slack-chat-api/internal/render"
	"github.com/open-cli-collective/slack-chat-api/internal/watch"
)

// NewCmd creates the watch command with its subcommands.
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Follow messages for replies and reactions as they happen",
		Long: `Follow messages for replies and reactions as Slack pushes them.

'slck watch add <ref>' follows a message: replies in its thread (from that
message on), reactions on any of them, edits and deletions. One listener,
'slck watch listen', holds a Socket Mode connection and keeps each watched
thread in a local file. 'status' and 'wait' read those files and never call
Slack, so any number of them can run at once.

A message ref is <channel_id>/<ts> or a permalink, as 'slck search' and
'slck messages send --permalink' print it.

Setup, once per Slack app (the app slck's user token belongs to):
  1. In the app's manifest set settings.socket_mode_enabled: true and
     settings.event_subscriptions.user_events to message.channels,
     message.groups, message.im, message.mpim, reaction_added and
     reaction_removed, and add the user scopes channels:history,
     groups:history, im:history, mpim:history and reactions:read
     ('slck apps update'). Reinstall the app so the user token gains the
     scopes.
  2. Generate an app-level token with the connections:write scope (the
     app's Basic Information page) and store it:
       <command that prints the token> | slck set-credential --key app_token --stdin
  3. Keep 'slck watch listen' running, for example as a launchd agent or a
     systemd user service.

Slack sends the listener every message and reaction in the conversations
the user is in; the listener keeps the ones on watched threads and drops
the rest. Each time it connects after being away, it reads each watched
thread once to catch up on what happened meanwhile.

Examples:
  slck watch add C02DF3BEUGN/1777469221.721439
  slck watch wait C02DF3BEUGN/1777469221.721439      # blocks until it changes
  slck watch status C02DF3BEUGN/1777469221.721439
  slck watch list
  slck watch remove C02DF3BEUGN/1777469221.721439`,
	}
	cmd.AddCommand(newAddCmd(), newListCmd(), newRemoveCmd(), newStatusCmd(), newWaitCmd(), newListenCmd())
	return cmd
}

func newAddCmd() *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "add <message-ref>",
		Short: "Follow a message for replies and reactions",
		Long: `Follow a message for replies in its thread and reactions, as the
listener receives them. Reads the thread once now, with the user token, so
'status' shows it at once and 'wait' sees only what comes after.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := watch.Open()
			if err != nil {
				return err
			}
			c, err := client.NewUserClient()
			if err != nil {
				return err
			}
			return runAdd(args[0], days, store, c, time.Now())
		},
	}
	cmd.Flags().IntVar(&days, "days", 14, "Stop following the message after this many days")
	return cmd
}

func runAdd(input string, days int, store *watch.Store, c *client.Client, now time.Time) error {
	ref, err := messageref.Parse(input)
	if err != nil {
		return err
	}
	if days < 1 {
		return fmt.Errorf("--days must be at least 1")
	}
	syncer := watch.NewSyncer(context.Background(), store, c, renderer(c), func() time.Time { return now }, os.Getpid(), logf)
	w, err := syncer.Add(ref, days)
	if err != nil {
		return err
	}
	output.Printf("Watching %s until %s.\n", ref, w.ExpiresAt.Local().Format("2006-01-02"))
	if l, err := store.Listener(); err == nil && !l.Live(now) {
		output.Printf("The listener is %s.\n", l.Describe(now))
	}
	return nil
}

// renderer renders a watch's messages as `messages read` prints them,
// resolving names with c.
func renderer(c *client.Client) watch.Render {
	resolver := client.NewUserResolver(c)
	return func(msgs []client.Message) string { return render.MessageList(msgs, resolver) }
}

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List followed messages and whether the listener is connected",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := watch.Open()
			if err != nil {
				return err
			}
			return runList(store, time.Now())
		},
	}
}

func runList(store *watch.Store, now time.Time) error {
	l, err := store.Listener()
	if err != nil {
		return err
	}
	output.Printf("Listener: %s\n", l.Describe(now))
	watches, err := store.List()
	if err != nil {
		return err
	}
	if len(watches) == 0 {
		output.Println("No messages watched.")
		return nil
	}
	output.Println()
	output.Println("REF | UNTIL | MESSAGES")
	for _, w := range watches {
		output.Printf("%s | %s | %d\n", w.Ref(), w.ExpiresAt.Local().Format("2006-01-02"), len(w.Messages))
	}
	return nil
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <message-ref>",
		Aliases: []string{"rm"},
		Short:   "Stop following a message",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := messageref.Parse(args[0])
			if err != nil {
				return err
			}
			store, err := watch.Open()
			if err != nil {
				return err
			}
			if err := store.Remove(ref); err != nil {
				return err
			}
			output.Printf("Stopped watching %s.\n", ref)
			return nil
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <message-ref>",
		Short: "Print a followed message's thread and reactions",
		Long: `Print a followed message's thread and reactions as the listener last
saw them, in the format of 'slck messages read'. Reads the local watch file
only. The output changes only when the thread does, so it suits a check
command that flags a change.

Fails when the listener is not connected, since the file may then be stale.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := watch.Open()
			if err != nil {
				return err
			}
			w, err := current(store, args[0], time.Now())
			if err != nil {
				return err
			}
			output.Printf("%s", w.Activity)
			return nil
		},
	}
}

// current loads a watch, refusing when the listener is not keeping it up to
// date.
func current(store *watch.Store, input string, now time.Time) (*watch.Watch, error) {
	ref, err := messageref.Parse(input)
	if err != nil {
		return nil, err
	}
	w, err := store.Load(ref)
	if errors.Is(err, watch.ErrNotWatched) {
		return nil, fmt.Errorf("%s is not watched; run `slck watch add %s`", ref, ref)
	}
	if err != nil {
		return nil, err
	}
	l, err := store.Listener()
	if err != nil {
		return nil, err
	}
	if !l.Live(now) {
		return nil, fmt.Errorf("the listener is %s", l.Describe(now))
	}
	return w, nil
}

func newWaitCmd() *cobra.Command {
	var timeout time.Duration
	cmd := &cobra.Command{
		Use:   "wait <message-ref>",
		Short: "Block until a followed message gets a reply, reaction or edit",
		Long: `Block until a followed message's thread changes, then print it and exit 0.
Reads the local watch file once a second and never calls Slack.

Exits 1 when the listener stops being connected, the watch is removed or
expires, or --timeout passes, so a waiting process never waits on a
connection that is gone. Run it in the background and act when it exits.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := watch.Open()
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runWait(ctx, args[0], timeout, time.Second, store, time.Now)
		},
	}
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "Give up after this long (for example 8h); 0 waits as long as the watch lasts")
	return cmd
}

func runWait(ctx context.Context, input string, timeout, every time.Duration, store *watch.Store, now func() time.Time) error {
	w, err := current(store, input, now())
	if err != nil {
		return err
	}
	baseline := w.Activity
	started := now()
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
		w, err := current(store, input, now())
		if err != nil {
			return err
		}
		if w.Activity != baseline {
			output.Printf("New activity on %s:\n%s", w.Ref(), w.Activity)
			return nil
		}
		if now().After(w.ExpiresAt) {
			return fmt.Errorf("the watch on %s expired with no new activity", w.Ref())
		}
		if timeout > 0 && now().Sub(started) >= timeout {
			return fmt.Errorf("no new activity on %s within %s", w.Ref(), timeout)
		}
	}
}

func newListenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "listen",
		Short: "Hold the Socket Mode connection that keeps watches up to date",
		Long: `Hold a Socket Mode connection with the app-level token and keep every
watched thread's file up to date. Runs until interrupted; reconnects when the
connection drops or Slack asks for a new one. Logs to standard error.

Only one listener runs at a time. Keep it running as a service (a launchd
agent, a systemd user service). See 'slck watch --help' for the app setup.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := watch.Open()
			if err != nil {
				return err
			}
			app, err := client.NewAppClient()
			if err != nil {
				return err
			}
			user, err := client.NewUserClient()
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return runListen(ctx, store, app, user)
		},
	}
}

func runListen(ctx context.Context, store *watch.Store, app, user *client.Client) error {
	if l, err := store.Listener(); err == nil && l.Live(time.Now()) && l.PID != os.Getpid() {
		return fmt.Errorf("a listener is already %s", l.Describe(time.Now()))
	}
	syncer := watch.NewSyncer(ctx, store, user, renderer(user), time.Now, os.Getpid(), logf)
	defer syncer.Close()
	// apps.connections.open is one short request under the client's 30s
	// timeout; Listen checks ctx around it.
	open := func(context.Context) (string, error) { return app.OpenSocketConnection() }
	return watch.Listen(ctx, open, syncer, logf)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "%s %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}
