# EVE Buddy

A multi-platform companion app for Eve Online players available on Windows, Linux, macOS and Android.

[![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy/releases)
[![Fyne](https://img.shields.io/badge/dynamic/regex?url=https%3A%2F%2Fgithub.com%2FErikKalkoken%2Fevebuddy%2Fblob%2Fmain%2Fgo.mod&search=fyne%5C.io%5C%2Ffyne%5C%2Fv2%20(v%5Cd*%5C.%5Cd*%5C.%5Cd*)&replace=%241&label=Fyne&cacheSeconds=https%3A%2F%2Fgithub.com%2Ffyne-io%2Ffyne)](https://github.com/fyne-io/fyne)
[![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ErikKalkoken/evebuddy)](https://goreportcard.com/report/github.com/ErikKalkoken/evebuddy)
[![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)](https://github.com/ErikKalkoken/evebuddy?tab=MIT-1-ov-file#readme)
[![chat](https://img.shields.io/discord/790364535294132234)](https://discord.gg/tVSCQEVJnJ)
[![GitHub Downloads](https://img.shields.io/github/downloads/ErikKalkoken/evebuddy/total)](https://tooomm.github.io/github-release-stats/?username=ErikKalkoken&repository=evebuddy)

[![download](https://github.com/user-attachments/assets/c8de336f-8c42-4501-86bb-dbc9c66db1f0)](https://github.com/ErikKalkoken/evebuddy/releases/latest)

## Contents

- [Description](#description)
- [Features](#features)
- [Screenshots](#screenshots)
- [Installing](#installing)
  - [Windows](#windows)
  - [macOS](#mac-os)
  - [Linux](#linux)
  - [Android](#android)
- [Updating](#updating)
- [Uninstalling](#uninstalling)
- [Support](#support)
- [FAQ](#faq)
- [External web sites](#external-web-sites)
- [Credits](#credits)

## Description

EVE Buddy is a multi-platform companion app for [Eve Online](https://www.eveonline.com/) players. It provides the following key features:

- **Character monitor**: Check current information about each of your characters, e.g. inspect the training queue of a character or browse it's assets.
- **Corporation monitor**: Check current information about each of your corporations: e.g. check corporation wallets or see all current members.
- **Overviews**: Keep track of and get unique insights about all your characters and corporations with consolidated views, e.g. find assets across of your characters or see which character has manufacturing slots available.
- **Notifications**: Get notified on your desktop or mobile about new EVE communications and other important updates, e.g. a structure was attacked or a training queue became empty.
- **New Eden search**: Search live on the game server, similar to in-game search bar, e.g. search for characters, corporations solar systems.
- **Information windows**: Show additional information for most objects on screen, similar to in-game information windows, e.g. sender of a mail:
- **Mail client**: Send and receive EVE mails for all your characters
- **Run in Background**: The app can run in the background and continue to notify you while you are doing something else, e.g. play Eve Online
- **Customizable UI**: UI can be customized, e.g. dark and light color theme

EVE Buddy is available for Windows, Linux, macOS and Android.

> [!Note]
> We are proud to have been mentioned twice in Eve Online's official Community Beat newsletter:
>
> - [EVE Community Beat newsletter 16 August 2025](https://www.eveonline.com/news/view/community-beat-for-16-august-2025).
> - [EVE Community Beat newsletter 22 November 2024](https://www.eveonline.com/news/view/community-beat-for-22-november).

> [!IMPORTANT]
> We would very much appreciate your feedback. If you encounter any problems or have a question please feel free to open an issue or come chat with us on our [Discord server]((https://discord.gg/tVSCQEVJnJ)) in the support channel **#evebuddy**.<br>
> EVE Buddy is in-development and we are constantly adding new features and improving the app further. If you are missing a feature that would make EVE Buddy more useful for you, please feel free to open a feature request.

> [!TIP]
> Help wanted! We would very much appreciate any contribution. If you like to provide a fix or add a feature please feel free top open a PR. Or if you have any questions please contact us on Discord.

## Features

The following is a detailed list of EVE Buddy's features. Most features are available for both desktop and mobile:

- **Overviews**: Keep track of and get unique insights about all your characters and corporations with consolidated views:
  - Assets: Search assets across all characters
  - Clones: Overview of all current clones and search nearest available jump clones across all characters
  - Colonies: Browse PI colonies across all characters
  - Contracts: Browse contracts of all characters
  - Industry: Browse industry jobs for all characters and related corporations
  - Location: Browse the location of all characters and their current ships
  - Training: Keep track of the training status for all characters
  - Wealth: Charts showing wealth distribution across all characters

- **Character monitor**: Check current information about each of your characters:
  - Assets: Browse through your assets at all your locations
  - Clones: Current augmentations, jump clones & jump cooldown timer
  - Mails: Browser through all mails
  - Communications: Browse through all communications
  - Skills: Training queue, catalogue of all trained skills and what ships can be flown
  - Wallet: Wallet and market Transactions

- **Corporation monitor**: Check current information about each of your corporations: (depending on their roles)
  - Members: List of current corporation members
  - Wallets: Wallet, market transactions and balances for corporation wallets

- **Notifications**: Get notified on your desktop or mobile about new EVE communications and other important updates:
  - Training queue became empty
  - Contract status changed
  - PI extraction went offline
  - New EVE communication received (e.g. structure attacked)
  - New EVE mail received

- **New Eden search**: Search live on the game server, similar to in-game search bar:
  - Search for: Agents, Alliances, Characters, Constellations, Corporations, Factions, Regions, Stations, Systems, Types
  - Simple or advanced search with filters
  - Search history

- **Information windows**: Show additional information for most objects on screen, similar to in-game information windows:
  - Alliances
  - Characters
  - Constellations
  - Corporations
  - Factions
  - Locations (i.e. structure or station)
  - Regions
  - Systems
  - Types

- **Mail client**: Full mail client for receiving and sending Eve mails

- **Run in Background**: The app can run in the background and continue to notify you while you are doing something else (e.g. play Eve Online)
  - Desktop: Can minimize to system tray and show an indicator for new EVE mail
  - Mobile: Will continue running in the background after switching to another app

- **Customizable UI**: The UI can be customized:
  - Color theme: Dark or Light theme
  - UI scaling: Custom scaling of the whole UI (desktop only)

## Screenshots

### Asset search across all characters

You can search for assets across all characters.

<img width="1920" height="1046" alt="desktop_asset_search" src="https://github.com/user-attachments/assets/c82a47eb-f3ae-4c14-b1b2-20f85b47507c" />

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

Home navigation | Training overview | Communications
-- | -- | ---
<img width="864" height="1920" alt="mobile_home" src="https://github.com/user-attachments/assets/43799cea-f164-4315-9a3b-6c99c86091d8" />|<img width="864" height="1920" alt="mobile_training" src="https://github.com/user-attachments/assets/7bf832f3-c0a0-43f0-905b-87e5ebdd88db" />|<img width="864" height="1920" alt="mobile_communications" src="https://github.com/user-attachments/assets/c7681b0d-8147-4031-867b-c9a01229ff2b" />

## Installing

To install EVE buddy just download the latest release from the releases page to your computer or mobile. The app ships as a single executable file that can be run directly. When you run the app for the first time it will automatically install itself for the current user (i.e. by creating folders in the home folder for the current user).

You find the latest packages for download on the [releases page](https://github.com/ErikKalkoken/evebuddy/releases).

### Windows

1. Download the windows zip file from the latest release on Github.
1. Unzip the file into a directory of your choice and run the .exe file to start the app.

> [!TIP]
> Windows defender may report EVE Buddy incorrectly as containing a trojan. This is a [known issue](https://github.com/microsoft/go/issues/1255) with programs made with the Go programming language. Also, each EVE Buddy release is build from scratch on a fresh Windows container on Github, so it is highly unlikely to be infected. If this happens to you, please exclude EVE Buddy's executable from Windows defender to proceed.

### Mac OS

1. Download the darwin zip file from the latest release on Github for your respective platform (arm or intel).
1. Unzip the file into a directory of your choice.
1. Run the .app file to start the app.

> [!TIP]
> MacOS may report EVE Buddy incorrectly as "damaged", because it is not signed with an Apple certificate. You can remove this error by opening a terminal and running the following command. For more information please see [Fyne Troubleshooting](https://docs.fyne.io/faq/troubleshoot#distribution):
>
> ```sudo xattr -r -d com.apple.quarantine "EVE Buddy.app"```

### Linux

We are providing two variants for installing on Linux desktop:

- AppImage: The AppImage variant allows you to run the app directly from the executable without requiring installation or root access.
- Tarball: The tar file requires installation, but also allows you to integrate the app into your desktop environment. The tarball also has wider compatibility among different Linux versions.

#### AppImage

> [!NOTE]
> The app is shipped in the [AppImage](https://appimage.org/) format, so it can be used without requiring installation and run on many different Linux distributions.

1. Download the latest AppImage file from the releases page
1. Make the AppImage file executable
1. Execute the AppImage file to start the app

> [!TIP]
> Should you get the following error: `AppImages require FUSE to run.`, you need to first install FUSE on your system. Thi s is a library required by all AppImages to function. Please see [this page](https://docs.appimage.org/user-guide/troubleshooting/fuse.html#the-appimage-tells-me-it-needs-fuse-to-run) for details.

#### Tarball

1. Download the latest tar file from the releases page
1. Decompress the tar file, for example with: `tar xf evebuddy-0.33.0-linux-amd64.tar.xz`
1. Run `make user-install` to install the app for the current user or run `sudo make install` to install the app on the system

You should now have a shortcut in your desktop environment's launcher for starting the app.

To uninstall the app again run either: `make user-uninstall` or `sudo make uninstall` depending on how you installed it.

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

> [!IMPORTANT]
> EVE Buddy needs unrestricted background usage in order to function probably. The reason is that Android otherwise  automatically suspends apps when you switch to another app. Then you can no longer add new characters, because it requires you to switch to your browser app, but EVE Buddy needs to keep running for the process to work. Also EVE Buddy needs to keep running in order to pick up events for notifications.
Please also make sure you do not have Power saving mode enabled (e.g. on Samsung Galaxy), which would also restrict background app usage.

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

On Android you can uninstall the app via the system's Settings app. This will also remove all data.

## Support

> [!IMPORTANT]
> For bugs and feature requests please open an issue in the GitHub repository.

> [!TIP]
> For support and any other questions please join us on in our channel #eve-buddy on this [Discord server](https://discord.gg/tVSCQEVJnJ).

### Common issues

The following is an overview of common issues with solutions:

#### Android

- [Timeout in browser when trying to add new character on some Android phones](https://github.com/ErikKalkoken/evebuddy/issues/76)

#### Windows

- [Issues with application scaling when moving between monitors of different resolutions](https://github.com/ErikKalkoken/evebuddy/issues/209)

### Logs

The app has an application log and a crash file that can help with trouble shooting. The location of the logs follows the standard of each platform:

Platform | Path
-- | --
Android | Export only
Linux | `/home/{username}/.local/share/evebuddy/log`
macOS | `/Users/{username}/Library/Application Support/evebuddy/log'`
Windows | `C:\Users\{username}\AppData\Local\evebuddy\evebuddy\log`

On desktop you can view the location of your log files on the User Data dialog, which you find in the main menu.

On both desktop and mobile you can export both logs from the Settings menu in the General section.

To view the exported logs on mobile you might want to install another app. While there are many decent apps for viewing log and text files on the Google Play store, we can recommend the following two apps:

- For viewing log files: [LogLog](https://play.google.com/store/apps/details?id=io.github.mthli.loglog&hl=en)
- For viewing txt files: [Text Viewer](https://play.google.com/store/apps/details?id=com.panagola.app.textviewer&hl=en)

## FAQ

### How well is my data protected?

EVE Buddy is designed to protect your data and tokens and has implemented the following safety measures:

1. All data and tokens retrieved from CCP's servers are stored on your local computer only. Your data is therefore safe as long as you prevent any unauthorized access to the data on your computer.

1. EVE Buddy also does not log any tokens (they are replaced with the text `REDACTED`). It is therefore safe to share your logs with maintainers for troubleshooting.

1. EVE Buddy is fully compliant with the requirements for [OAuth 2.0 for Mobile or Desktop Applications](https://docs.esi.evetech.net/docs/sso/native_sso_flow.html) from CCP.

1. In case you need to switch computers you can remove your data with the [delete app](#uninstalling).

### Why do I not see all of my character's data in the app?

Some of your data from the game server might not be visible in EVE Buddy due to technical limitations of the game server API (ESI) or missing permissions.

#### Server limitations

CCP's servers have limitations on how far back some character data can be retrieved.

Here is an overview of some limitations:

- Wallet journal: 30 days, 2.500 entries
- Wallet transaction: 2.500 entries

 However, EVE Buddy will keep all historic data once retrieved. For example: If you allow EVE Buddy to update on a regular basis, it will be able to keep a record of your wallet transactions over many months and years.

#### Permissions

A special case are Upwell structures. Access to structures depends on in-game docking rights. Unfortunately, it is not possible to later retrieve the name or location of a structure, which the character no longer has access to. For example character assets might be displayed in an "unknown structure".

## External web sites

You can find EVE Buddy mentions on other web sites:

- [EVE Forum](https://forums.eveonline.com/t/eve-buddy-a-companion-app-for-desktop-and-mobile-v0-40)
- [EVE Developer site](https://developers.eveonline.com/docs/community/evebuddy/)
- [Reddit: Initial](https://www.reddit.com/r/Eve/comments/1go4iee/eve_buddy_a_new_companion_app_for_eve_online/)
- [Reddit: Update](https://www.reddit.com/r/Eve/comments/1mjwn79/update_eve_buddy_a_companion_app_for_eve_online/)

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
