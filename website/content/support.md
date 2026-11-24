---
title: Support
lead: Answers to common questions, known issues and where to get help.
description: Frequently asked questions and support for EVE Buddy.
toc: true
---

## Getting help

For bugs and feature requests please [open an issue on GitHub](https://github.com/ErikKalkoken/evebuddy/issues).

For support and any other questions please join us in the **#eve-buddy** channel on our [Discord server](https://discord.gg/tVSCQEVJnJ).

## FAQ

### How well is my data protected?

EVE Buddy is designed to protect your data and tokens and has implemented the following safety measures:

1. All data and tokens retrieved from CCP's servers are stored on your local computer only. Your data is therefore safe as long as you prevent any unauthorized access to the data on your computer.
1. EVE Buddy does not log any tokens (they are replaced with the text `REDACTED`). It is therefore safe to share your logs with maintainers for troubleshooting.
1. EVE Buddy is fully compliant with the requirements for [OAuth 2.0 for Mobile or Desktop Applications](https://docs.esi.evetech.net/docs/sso/native_sso_flow.html) from CCP.
1. In case you need to switch computers you can remove your data with the [delete option](install/#uninstalling).

### When I add a character the app redirects to localhost. Shouldn't it redirect to a CCP server?

No, the redirect to localhost is correct. It connects to EVE Buddy's own web server, which is needed for CCP's authorization process called SSO. For more information please see the [EVE Developer Documentation / SSO](https://developers.eveonline.com/docs/services/sso/).

### Why do I not see all of my character's data in the app?

Some of your data from the game server might not be visible in EVE Buddy due to technical limitations of the game server API (ESI) or missing permissions.

**Server limitations**: CCP's servers limit how far back some character data can be retrieved:

- Wallet journal: 30 days, 2,500 entries
- Wallet transactions: 2,500 entries

However, EVE Buddy keeps all historic data once retrieved. For example, if you allow EVE Buddy to update on a regular basis, it will keep a record of your wallet transactions over many months and years.

**Permissions**: A special case are Upwell structures. Access to structures depends on in-game docking rights. Unfortunately, it is not possible to later retrieve the name or location of a structure the character no longer has access to. For example, character assets might be displayed in an "unknown structure".

## Common issues

### Android

- [Timeout in browser when trying to add a new character on some Android phones](https://github.com/ErikKalkoken/evebuddy/issues/76)

### Windows

- [Issues with application scaling when moving between monitors of different resolutions](https://github.com/ErikKalkoken/evebuddy/issues/209)

## Logs

The app has an application log and a crash file that can help with troubleshooting. The location of the logs follows the standard of each platform:

| Platform | Path |
| -- | -- |
| Android | Export only |
| Linux | `/home/{username}/.local/share/evebuddy/log` |
| macOS | `/Users/{username}/Library/Application Support/evebuddy/log` |
| Windows | `C:\Users\{username}\AppData\Local\evebuddy\evebuddy\log` |

On desktop you can see the location of your log files in the User Data dialog, which you find in the main menu.

On both desktop and mobile you can export both logs from the Settings menu in the General section.

To view exported logs on mobile you might want to install another app. While there are many decent apps for viewing log and text files on the Google Play store, we can recommend these two:

- For viewing log files: [LogLog](https://play.google.com/store/apps/details?id=io.github.mthli.loglog&hl=en)
- For viewing txt files: [Text Viewer](https://play.google.com/store/apps/details?id=com.panagola.app.textviewer&hl=en)
