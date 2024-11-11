# EVE Buddy

A companion app for Eve Online players available on Windows, Linux and macOS.

[![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy/releases)
[![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml)
[![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy?tab=MIT-1-ov-file#readme)
[![chat](https://img.shields.io/discord/790364535294132234)](https://discord.gg/tVSCQEVJnJ)

## Contents

- [Description](#description)
- [Highlights](#highlights)
- [Installing](#installing)
- [Updating](#updating)
- [Uninstalling](#uninstalling)
- [FAQ](#faq)
- [Credits](#credits)

## Description

EVE Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. It has three key features:

- Give you access to your characters without having to log into the Eve client or switching your current Eve character
- Provide you with helpful functions, that the Eve client is lacking, like asset search across all your characters
- Notify you about important game events (e.g. structure attacked) and new Eve mails

> [!IMPORTANT]
> This is an early version and not yet considered fully stable. We would very much appreciate your feedback, so we can find and squash remaining bugs. If you encounter any problems please feel free to open an issue or chat with us on our [Discord server]((https://discord.gg/tVSCQEVJnJ)) in the support channel **#evebuddy**.<br>
> Some features may not be fully implemented yet (e.g. some notification types). Our current focus is on bug fixing, rather then adding more features. But if you are missing anything important, please feel free to open a feature request.<br>
> We very much welcome any contributions. If you like to provide a fix or add a feature please feel free top open a PR.

## Features

A more detailed overview of the provided features:

- Information about each character:
  - Assets: Full asset browser
  - Character: Curren clone, jump clones, and more...
  - Mails: Full mail client for receiving and sending Eve mails
  - Skills: Training queue, catalogue of all trained skills and what ships can be flown
  - Wallet: Wallet and Wallet Transactions
- Overview of all your characters (e.g. wallet, skill points, location)
- Wealth: Charts showing wealth distribution across all characters
- Assets search: Full asset search across all your characters
- Can minimize to system tray and show indicator for new mail
- Receive desktop notifications about new communications (e.g. Structure gets attacked) and mails
- Single executable file, no installation required
- Desktop app that runs on Windows, Linux and macOS
- Automatic dark and light theme
- Offline mode

## Highlights

### Asset browser for each character

You can browse the assets of a character by location. The view by location is similar to the in-game view when docked.

![Screenshot from 2024-11-10 16-53-30](https://github.com/user-attachments/assets/9c4991ab-406a-44cd-9a18-fec1c10c1a42)

### Full mail client

You can receive, send and delete eve mails. Similar to the in-game mail client.

![Screenshot from 2024-11-10 16-52-55](https://github.com/user-attachments/assets/d7b226c5-1355-4b99-bef7-e0a3b1e75cd6)

### Overview of all characters

The overview pages gives you key information about all you characters at a glance.

![Screenshot from 2024-11-10 16-54-22](https://github.com/user-attachments/assets/10838273-3a75-4160-aabf-5cc895bde1c4)

### Asset search across all characters

You can search for assets across all characters.

![Screenshot from 2024-11-10 16-52-07](https://github.com/user-attachments/assets/c40f3b7f-279f-4b3c-9135-0c3b043ee0d9)

### Wealth charts across all characters

The wealth page gives you a graphical overview of your total wealth (= wallets + asset value) and contains breakdowns that help you better understand the structure of your wealth.

![Screenshot from 2024-11-10 16-54-56](https://github.com/user-attachments/assets/3d26e44f-cdc7-45fe-a441-4ed982662fa7)

### Desktop notifications

EVE Buddy can send your desktop notifications to inform you about new communications and mails.

![Screenshot from 2024-11-10 17-18-27](https://github.com/user-attachments/assets/0a05ddec-bf31-42c6-a1f1-c2661bd12c49)

## Installing

To install EVE buddy just download and unzip the latest release from the releases page to your computer. The app ships as a single executable file that can be run directly. When you run the app for the first time it will automatically install itself for the current user (i.e. by creating folders in the home folder for the current user).

You find the latest packages for download on the [releases page](https://github.com/ErikKalkoken/evebuddy/releases).

### Linux

> [!NOTE]
> The app is shipped in the [AppImage](https://appimage.org/) format, so it can be used without requiring installation and run on many different Linux distributions.

1. Download the latest AppImage file from the releases page and make it executable.
1. Execute it to start the app.

> [!TIP]
> Should you get the following error: `AppImages require FUSE to run.`, you need to first install FUSE on your system. Thi s is a library required by all AppImages to function. Please see [this page](https://docs.appimage.org/user-guide/troubleshooting/fuse.html#the-appimage-tells-me-it-needs-fuse-to-run) for details.

### Windows

1. Download the windows zip file from the latest release on Github.
1. Unzip the file into a directory of your choice and run the .exe file to start the app.

### Mac OS

1. Download the darwin zip file from the latest release on Github for your respective platform (arm or intel).
1. Unzip the file into a directory of your choice
1. Run the .app file to start the app.

> [!TIP]
> MacOS may report this app incorrectly as "damaged", because it is not signed with an Apple certificate. You can remove this error by opening a terminal and running the following command. For more information please see [Fyne Troubleshooting](https://docs.fyne.io/faq/troubleshoot#distribution):
>
> ```sudo xattr -r -d com.apple.quarantine "EVE Buddy.app"```

### Build and run from repository

You can also build and run the app directly from the repository. For that your system needs to be able to build Fyne apps, which requires you to have installed the Go tools, a C compiler and a systems graphics driver. For details please see [Fyne - Getting started](https://docs.fyne.io/started/).

When you have all necessary tools installed, you can build and run this app direct from the repository with:

```sh
go run github.com/ErikKalkoken/evebuddy@latest
```

## Updating

The app will inform you when there is a new version available for download. To update your app just download and unzip the newest version for your platform from the [releases page](https://github.com/ErikKalkoken/evebuddy/releases). Then overwrite the old executable file with the new one.

## Uninstalling

If you no longer want to use the app you can uninstall it.

First start the delete app for removing your user data:

```sh
./evebuddy --delete-data
```

This command will ask for confirmation and then delete user data from your computer like characters, log files, etc.

Then delete the file itself:

```sh
rm evebuddy
```

## FAQ

### Where can I get support?

Fo bugs and feature requests please open an issue in the GitHub repository.

For support and any other questions please join us on in our channel #eve-buddy on this [Discord server](https://discord.gg/tVSCQEVJnJ).

### What safety measures are taken to protect my character data and token?

EVE Buddy is designed to protect your character data and token and has implemented the following safety measures:

1. All character data and tokens retrieved from CCP's servers are stored on your local computer only. Your data is therefore safe as long as you prevent any unauthorized access to the data on your computer.

1. EVE Buddy also does not log any tokens (they are replaced with the text `REDACTED`). It is therefore safe to share your logs with maintainers for troubleshooting.

1. EVE Buddy is fully compliant with the requirements for [OAuth 2.0 for Mobile or Desktop Applications](https://docs.esi.evetech.net/docs/sso/native_sso_flow.html) from CCP.

1. In case you need to switch computers you can remove your data with the [delete app](#uninstalling).

### Why do I not see all of my character's data in the app?

#### Server limitations

CCP's servers have limitations on how far back some character data can be retrieved.

Here is an overview of some limitations:

- Wallet journal: 30 days, 2.500 entries
- Wallet transaction: 2.500 entries

 However, EVE Buddy will keep all historic data once retrieved. For example: If you allow EVE Buddy to update on a regular basis, it will be able to keep a record of your wallet transactions over many months and years.

#### Structures

A special case are Upwell structures. Access to structures depends on in-game docking rights. Unfortunately, it is not possible to later retrieve the name or location of a structure, which the character no longer has access to. For example character assets might be displayed in an "unknown structure".

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
