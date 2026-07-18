package commands

import (
	"encoding/json"
	"fmt"

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
		mailCmd.AddCommand(newMailListCmd(d), newMailGetCmd(d))
		return mailCmd
	})
}

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
