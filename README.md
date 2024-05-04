# Eve Buddy

Eve Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. It runs on Windows, MacOS and LINUX desktops.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)
![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)

This app is in development.

## Features

- Mails: Full client for Eve Mails
- Character stats: Display of current information about a character (e.g. location)

More features planned...

## Installation

### Linux

First download the linux tar file from the latest release on Github.

Then install the release on your desktop with:

```sh
sudo tar xvfJ evebuddy-v1.0.0-linux-amd64.tar.xz -C /
```

This will install the app for all users on your system. User specific data will be stored in the home directories of each user.

### Windows

First download the windows zip file from the latest release on Github.

Then unzip the file into a directory of your choice and run the .exe file to start the app.

### Mac OS

First download the darwin zip file from the latest release on Github.

Then unzip the file into a directory of your choice and run the .app file to start the app.

## Native Go

If your system is setup to run and compile fyne apps in Go you can start the app directly from the repp with:

```sh
go run github.com/ErikKalkoken/evebuddy@latest
```

To setup the necessary local development environment for fyne apps please follow the platform specific instructions in the [fye docs](https://docs.fyne.io/started/).

## Privacy notes

We understand and respect the privacy concerns of our fellow Eve Online players. Therefore all data of this app is stored locally (e.g. character tokens) and not shared with any 3rd party. Internet requests by this app are made to CCP's game server only.

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
