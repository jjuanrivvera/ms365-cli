# ms365

A fast, scriptable CLI for **Microsoft 365** via the **Microsoft Graph API**: Outlook
mail, calendar, and identity from your terminal — with named accounts so a personal
Outlook.com sign-in and a work/school tenant live side by side.

```sh
ms365 auth login -a personal
ms365 mail list --folder inbox --search "invoice"
ms365 calendar events --from 2026-07-20 --to 2026-07-27
```

See [Getting started](getting-started.md) and the [command reference](commands/ms365.md).
