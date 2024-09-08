# EVE Buddy

A companion app for Eve Online players

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)
![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)

## Contents

- [Description](#description)
- [Screenshot](#screenshot)
- [Installation](#installation)
- [Updating](#updating)
- [Removing the app](#removing-the-app)
  [FAQ](#faq)
- [Credits](#credits)

## Description

EVE Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. It has three key features:

- Give you access to your characters without having to log into the Eve client or switching your current Eve character
- Provide you with helpful functions, that the Eve client is lacking, like asset search across all your characters
- Notify you about game important events (e.g. structure attacked) and new Eve mails

> [!IMPORTANT]
> This is an early version and not yet considered fully stable. We would very much appreciate your feedback, so we can find and squash remaining bugs. If you encounter any problems please feel free to open an issue or chat with us on Discord.
> Some features may not be fully implemented yet (e.g. some notification types). Our current focus is on bug fixing, rather then adding more features. But if you are missing anything important, please feel free to open a feature request.
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

## Screenshot

![example](https://cdn.imgpile.com/f/aD27GDt_xl.png)

## Installation

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

## Removing the app

If you no longer want to use the app, here is how you can uninstall it from your computer.

First run the uninstall command to delete all data (example are for Linux):

```sh
./evebuddy -uninstall
```

Then delete the file itself:

```sh
rm evebuddy
```

## Troubleshooting

The app can be started with optional command line arguments, which offers some additional features and can help with trouble shooting. For example you can enable logging to a file and/or increase the log level.

For a description of all features please run the app with the help flag:

```sh
./evebuddy -h
```

## FAQ

### Where is my data stored? I am concerned about potentially leaking sensitive data

All data downloaded from CCP's servers is stored on your computer only. There is no data transferred to any servers and the maintainers of this software have no access to your data. Therefore there is very little risk of sensitive data being leaked.

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
