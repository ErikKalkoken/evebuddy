# EVE Buddy

A companion app for Eve Online players available on Windows, Linux, macOS and Android.

[![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy/releases)
[![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml)
[![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy?tab=MIT-1-ov-file#readme)
[![chat](https://img.shields.io/discord/790364535294132234)](https://discord.gg/tVSCQEVJnJ)
[![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/ErikKalkoken/evebuddy/total)](https://github.com/ErikKalkoken/evebuddy/releases)

## Contents

- [Description](#description)
- [Highlights](#highlights)
- [Installing](#installing)
  - [Linux](#linux)
  - [Windows](#windows)
  - [macOS](#mac-os)
  - [Android](#android)
  - [From source](#from-source)
- [Updating](#updating)
- [Uninstalling](#uninstalling)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)
- [Credits](#credits)

## Description

EVE Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players available for desktop and mobile. It has three key features:

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
  - Assets: Browse through your assets at all your locations
  - Clones: Current augmentations, jump clones & jump cooldown timer
  - Mails: Full mail client for receiving and sending Eve mails
  - Communications: Browse through all communications
  - Skills: Training queue, catalogue of all trained skills and what ships can be flown
  - Wallet: Wallet and Wallet Transactions
- Combined view over all characters:
  - Assets: Assets of all characters with full text search feature
  - Clones: Jump clones of all characters with route search from any location 
  - Colonies: Show list of all planetary industry colonies
  - Contracts: Active and history of contracts
  - Industry: Active and history of industry jobs
  - Location: Location in New Eden and current ships for all characters
  - Training: Queue status and skillpoints
  - Wealth: Charts showing wealth distribution across all characters
- Get notified about important events (e.g. structure attacked, training queue expired)
- Ability to search New Eden (similar to in-game search bar)
- Show information about various entities (e.g. character, alliance, location); similar to in-game information windows
- Can minimize to system tray and show indicator for new mail (desktop only)
- Available for desktop (Windows, macOS, Linux) and mobile (Android)
- Automatic dark and light theme
- Offline mode

## Highlights

### Asset search across all characters

You can search for assets across all characters.

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/qN4Wj7K_xl.png)

### Notifications

EVE Buddy can send notifications on your local device to notify about new communications, mails, expired training queues and more.

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/WFsQGDV_xl.png)

### Industry jobs

See active industry jobs and the history of industry jobs for all characters in one combined view.

![Screenshot from 2025-04-10 23-45-24](https://github.com/user-attachments/assets/b5071430-1301-447f-b7a6-054c64f86d90)

### Jump clones

See available jump clones and when the next clone jump is available for a character:

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/7kCXcj5_xl.png)

### New Eden search

You search the live game server for entities, similar to the in-game search bar:

![Screenshot from 2025-03-17 22-10-43](https://github.com/user-attachments/assets/b182033d-9f03-447b-a812-e1ad363f0893)

### Info windows

Access additional information about many entities (e.g. characters, corporations, solar systems):

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/japrVbC_xl.png)

### Full mail client

You can receive, send and delete eve mails. Similar to the in-game mail client.

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/cjp9mUs_xl.png)

### Asset browser for each character

You can browse the assets of a character by location. The view by location is similar to the in-game view when docked.

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/5vYS17c_xl.png)

### Colonies of all characters

You can see all colonies used for planetary industry from all your characters at a glance.

![Screenshot from 2025-04-03](https://cdn.imgpile.com/f/xFnr2yP_xl.png)

### Mobile version

EVE Buddy also works on Android and has a mobile friendly navigation:

Character navigation | Mail browser | Asset browser
-- | -- | --
![Screenshot_20250301-001409](https://github.com/user-attachments/assets/05b4d70c-65b5-49c6-95ab-d2a6dff846af)|![Screenshot_20250301-002237](https://github.com/user-attachments/assets/d8eec777-2690-49e6-8def-c38ee91ca81e)|![Screenshot_20250301-002825](https://github.com/user-attachments/assets/415956cd-c1d1-471d-afa5-51ff8034dbce)

## Installing

To install EVE buddy just download the latest release from the releases page to your computer or mobile. The app ships as a single executable file that can be run directly. When you run the app for the first time it will automatically install itself for the current user (i.e. by creating folders in the home folder for the current user).

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

### Android

> [!NOTE]
> Please note that the app can currently be only installed from the release page and is currently not available in any of the Android app stores.

1. Make sure you have enabled in settings that you can install from your web browser (For detailed instructions please see [How to install Unknown Sources applications in Android](https://www.appaloosa.io/blog/guides/how-to-install-apps-from-unknown-sources-in-android))
1. Navigate to the github releases page in your mobile browser
1. Select to download the latest EVE_Buddy.apk file from the release page (this can take a minute)
1. When prompted choose to install the file / open the file with the default installer
1. In case you get a security warning from Google Play Protect:
   1. Select "More details"
   1. Select "Install anyway"
1. Enable unrestricted background usage for EVE Buddy in settings. For a guide please see [How to prevent apps from 'sleeping' in the background on Android](https://www.androidpolice.com/prevent-apps-from-sleeping-in-the-background-on-android/)
1. Select "Unrestricted" under "App battery usage" / "App background usage"

To enable notifications:

1. Go to settings and enable what kind of notifications you want to receive (e.g. mails)
1. Send a test notification (via menu in settings)
1. When asked: Allow EVE buddy to send Android notifications

You should now see a test notification.

> [!NOTE]
> EVE Buddy needs unrestricted background usage in order to function probably. The reason is that Android otherwise  automatically suspends apps when you switch to another app. Then you can no longer add new characters, because it requires you to switch to your browser app, but EVE Buddy needs to keep running for the process to work. Also EVE Buddy needs to keep running in order to pick up events for notifications.

### From source

It is also possible to build and run the app directly from the source on the github repository. For that to work your system needs to be setup for building Fyne apps, which requires you to have installed the Go tools, a C compiler and a systems graphics driver. For details please see [Fyne - Getting started](https://docs.fyne.io/started/).

When you have all necessary tools installed, you can build and run this app direct from the repository with:

```sh
go run github.com/ErikKalkoken/evebuddy@latest
```

## Updating

The app will inform you when there is a new version available for download. To update your app just download and install the newest version for your platform from the [releases page](https://github.com/ErikKalkoken/evebuddy/releases).

## Uninstalling

If you no longer want to use the app you can uninstall it.

### Windows, Linux and macOS

The desktop versions has an special app for removing our data:

First start the delete app for removing your user data:

```sh
./evebuddy --delete-data
```

This command will ask for confirmation and then delete user data from your computer like characters, log files, etc.

Then delete the file itself:

```sh
rm evebuddy
```

### Android

On Android you can uninstall the app via Android Settings and it will also remove all data.

## Troubleshooting

The app has an application log and a crash file that can help with trouble shooting. You can export both logs from the Settings menu in the General section.

To view the logs on mobile you might want to install another app. While there are many decent apps for viewing log and text files on the Google Play store, we can recommend the following two apps:

- For viewing log files: [LogLog](https://play.google.com/store/apps/details?id=io.github.mthli.loglog&hl=en)
- For viewing txt files: [Text Viewer](https://play.google.com/store/apps/details?id=com.panagola.app.textviewer&hl=en)

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
