# EVE Buddy

EVE Buddy is a companion app for Eve Online players. It runs on Windows, MacOS and LINUX desktops.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)
![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)

## Contents

- [Description](#description)
- [Screenshot](#screenshot)
- [How to Run](#how-to-run)
  [Data Privacy](#data-privacy)
- [Credits](#credits)

## Description

EVE Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. Key features are:

- Import information for each characters:
  - Assets: Full asset browser
  - Character: Curren clone, jump clones, and more...
  - Mails: Full mail client for receiving and sending Eve mails
  - Skills: Training queue, catalogue of all trained skills and what ships can be flown
  - Wallet: Wallet and Wallet Transactions
- Overview of all your characters (e.g. wallet, skill points, location)
- Wealth: Charts showing wealth distribution across all characters
- Assets search: Full asset search across all your characters

## Screenshot

![example](https://cdn.imgpile.com/f/aD27GDt_xl.png)

## How to run

To run EVE buddy just download and unzip the latest release to your computer. The app ships as a single executable file that can be run directly. You find the latest packages for download on the [releases page](https://github.com/ErikKalkoken/evebuddy/releases).

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

> [!NOTE]
> The MAC version is currently experimental only, since we have not been able to verify that the release process actually works. We would very much appreciate any feedback on wether the package works or what needs to be improved.

1. Download the darwin zip file from the latest release on Github.
1. Unzip the file into a directory of your choice
1. Run the .app file to start the app.

### Build and run from repository

You can also build and run the app directly from the repository. For that your system needs to be able to build Fyne apps, which requires you to have installed the Go tools, a C compiler and a systems graphics driver. For details please see [Fyne - Getting started](https://docs.fyne.io/started/).

When you have all necessary tools installed, you can build and run this app direct from the repository with:

```sh
go run github.com/ErikKalkoken/evebuddy@latest
```

## Data privacy

We understand and respect the privacy concerns of our fellow Eve Online players. Therefore all data of this app is stored and kept locally (e.g. character tokens and data). Internet requests by this app are made to CCPs game servers only.

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
