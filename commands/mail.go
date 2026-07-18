package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/jjuanrivvera/ms365-cli/internal/api"
	"github.com/jjuanrivvera/ms365-cli/internal/output"
)

// mailListColumns are the default table columns for message listings.
var mailListColumns = []string{"from.emailAddress.address", "subject", "receivedDateTime", "id"}

func init() {
	registrars = append(registrars, func(d *deps) *cobra.Command {
		mailCmd := &cobra.Command{
			Use:     "mail",
			Aliases: []string{"messages"},
			Short:   "Read Outlook mail (Mail.Read)",
		}
		mailCmd.AddCommand(newMailListCmd(d), newMailGetCmd(d), newMailSendCmd(d), newMailReplyCmd(d))
		return mailCmd
	})
}

// scopeMailSend is the delegated scope mail send/reply need beyond DefaultScopes.
const scopeMailSend = "Mail.Send"

func newMailListCmd(d *deps) *cobra.Command {
	var opts api.MailListOptions
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages",
		Long: `List messages from the mailbox (newest first). --folder accepts a well-known name
(inbox, sentitems, drafts, deleteditems, junkemail, archive) or a folder id.

--search is Exchange KQL ($search); --filter is OData ($filter). Microsoft Graph
rejects combining them, and $search results ignore ordering.`,
		Example: `  ms365 mail list --top 20
  ms365 mail list --folder inbox --search "from:ana subject:invoice"
  ms365 mail list --filter "isRead eq false" -o json
  ms365 mail list -a work --all --limit 200 -o csv`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, _, err := d.getAPIClient()
			if err != nil {
				return err
			}
			opts.Limit = d.gf.limit
			opts.All = d.gf.all
			items, err := c.MailList(cmd.Context(), opts)
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			return d.render(cmd, rawItems(items), mailListColumns)
		},
	}
	cmd.Flags().StringVar(&opts.Folder, "folder", "", "mail folder: well-known name (inbox, sentitems, …) or id")
	cmd.Flags().IntVar(&opts.Top, "top", 0, "page size requested from Graph ($top, max 1000)")
	cmd.Flags().StringVar(&opts.Search, "search", "", "KQL search query ($search); not combinable with --filter")
	cmd.Flags().StringVar(&opts.Filter, "filter", "", "OData filter ($filter), e.g. \"isRead eq false\"")
	cmd.Flags().StringVar(&opts.Select, "select", "", "override the $select field list")
	return annotate(cmd, kindRead)
}

func newMailGetCmd(d *deps) *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "get <message-id>",
		Short: "Show one message with its body as text",
		Long: `Fetch a single message. By default the body is down-converted to plain text and
printed in a readable letter form; --raw keeps the original (usually HTML) body.
Use -o json for the complete Graph resource.`,
		Example: `  ms365 mail get AAMkAGI2…
  ms365 mail get AAMkAGI2… --raw -o json
  ms365 mail list --top 1 -o id | xargs ms365 mail get`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, err := d.getAPIClient()
			if err != nil {
				return err
			}
			msg, body, err := c.MailGet(cmd.Context(), args[0], !raw)
			if err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			// Structured formats get the full Graph resource; the default view is a
			// readable letter (headers + text body).
			if d.gf.outputFormat != "" && d.gf.outputFormat != "table" {
				return d.render(cmd, body, nil)
			}
			printMessage(cmd, msg)
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "keep the original body (HTML) instead of text down-conversion")
	return annotate(cmd, kindRead)
}

func newMailSendCmd(d *deps) *cobra.Command {
	var opts api.MailSendOptions
	var bodyFile string
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send an email (Mail.Send)",
		Long: `Compose and send a message via /me/sendMail. The body is plain text by default;
--html marks it as HTML. --body-file reads the body from a file ("-" for stdin).

Sending needs the Mail.Send delegated scope, which the default sign-in does NOT
request — grant it once per account with:
  ms365 auth login -a <account> --scopes Mail.Send`,
		Example: `  ms365 mail send --to ana@example.com --subject "Lunch?" --body "12:30 at the usual place"
  ms365 mail send --to a@x.com,b@x.com --cc boss@x.com --subject "Report" --body-file report.txt
  ms365 mail send --to ops@x.com --subject "Alert" --html --body "<b>disk full</b>" --save-to-sent=false`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := resolveBodyFlags(cmd, opts.Body, bodyFile)
			if err != nil {
				return err
			}
			opts.Body = body
			c, _, err := d.getAPIClientScoped(scopeMailSend)
			if err != nil {
				return err
			}
			if err := c.MailSend(cmd.Context(), opts); err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Sent %q to %s.\n", opts.Subject, strings.Join(opts.To, ", "))
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&opts.To, "to", nil, "recipient address (repeatable or comma-separated)")
	cmd.Flags().StringSliceVar(&opts.Cc, "cc", nil, "CC address (repeatable or comma-separated)")
	cmd.Flags().StringSliceVar(&opts.Bcc, "bcc", nil, "BCC address (repeatable or comma-separated)")
	cmd.Flags().StringVar(&opts.Subject, "subject", "", "message subject")
	cmd.Flags().StringVar(&opts.Body, "body", "", "message body text")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `read the body from this file ("-" for stdin)`)
	cmd.Flags().BoolVar(&opts.HTML, "html", false, "send the body as HTML instead of plain text")
	cmd.Flags().BoolVar(&opts.SaveToSent, "save-to-sent", true, "keep a copy in Sent Items")
	// Sending mail is irreversible — there is no unsend on the wire.
	return annotate(cmd, kindDestructive)
}

func newMailReplyCmd(d *deps) *cobra.Command {
	var body, bodyFile string
	var all bool
	cmd := &cobra.Command{
		Use:   "reply <message-id>",
		Short: "Reply to a message (Mail.Send)",
		Long: `Reply to an existing message via /me/messages/{id}/reply — Exchange quotes the
original thread server-side and addresses the sender for you. --all replies to
every recipient (/replyAll).

Replying needs the Mail.Send delegated scope, which the default sign-in does NOT
request — grant it once per account with:
  ms365 auth login -a <account> --scopes Mail.Send`,
		Example: `  ms365 mail reply AAMkAGI2… --body "Works for me, see you then."
  ms365 mail reply AAMkAGI2… --all --body-file answer.txt
  ms365 mail list --top 1 -o id | xargs -I{} ms365 mail reply {} --body "ack"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			comment, err := resolveBodyFlags(cmd, body, bodyFile)
			if err != nil {
				return err
			}
			if comment == "" {
				return fmt.Errorf("a reply body is required — pass --body or --body-file")
			}
			c, _, err := d.getAPIClientScoped(scopeMailSend)
			if err != nil {
				return err
			}
			if err := c.MailReply(cmd.Context(), args[0], comment, all); err != nil {
				return err
			}
			if d.gf.dryRun {
				return nil
			}
			verb := "sender"
			if all {
				verb = "all recipients"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Replied to %s of message %s.\n", verb, args[0])
			return nil
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "reply body text")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", `read the reply body from this file ("-" for stdin)`)
	cmd.Flags().BoolVar(&all, "all", false, "reply to all recipients (/replyAll) instead of just the sender")
	return annotate(cmd, kindWrite)
}

// resolveBodyFlags merges --body and --body-file into one body string. --body-file wins
// over an empty --body; passing both is a user mistake worth surfacing, not merging.
func resolveBodyFlags(cmd *cobra.Command, body, bodyFile string) (string, error) {
	if bodyFile == "" {
		return body, nil
	}
	if body != "" {
		return "", fmt.Errorf("--body and --body-file are mutually exclusive")
	}
	b, err := readDataArg(cmd, fileArg(bodyFile))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// fileArg converts a --body-file value into readDataArg's convention ("-" stays stdin,
// anything else is an @file reference).
func fileArg(path string) string {
	if path == "-" {
		return "-"
	}
	return "@" + path
}

// printMessage renders one message like a mail reader would: headers then body, with
// terminal escapes stripped from API-controlled text.
func printMessage(cmd *cobra.Command, m *api.Message) {
	out := cmd.OutOrStdout()
	san := output.SanitizeTerminal
	fmt.Fprintf(out, "From:     %s <%s>\n", san(m.From.EmailAddress.Name), san(m.From.EmailAddress.Address))
	for i, r := range m.ToRecipients {
		label := "To:      "
		if i > 0 {
			label = "         "
		}
		fmt.Fprintf(out, "%s %s <%s>\n", label, san(r.EmailAddress.Name), san(r.EmailAddress.Address))
	}
	fmt.Fprintf(out, "Date:     %s\n", san(m.ReceivedDateTime))
	fmt.Fprintf(out, "Subject:  %s\n", san(m.Subject))
	if m.HasAttachments {
		fmt.Fprintln(out, "Attachments: yes")
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, san(m.Body.Content))
}

// rawItems normalizes a Graph collection into one JSON array for the renderer.
func rawItems(items []json.RawMessage) json.RawMessage {
	if items == nil {
		return json.RawMessage("[]")
	}
	b, _ := json.Marshal(items)
	return b
}
