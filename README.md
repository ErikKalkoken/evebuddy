# EVE Buddy

EVE Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. It runs on Windows, MacOS and LINUX desktops.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/evebuddy)
![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/ci-cd.yml/badge.svg)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/evebuddy)

**!! This app is currently in development and not ready yet for production use !!**

## Contents

- [Features](#features)
- [Screenshot](#screenshot)
- [Installation](#installation)
  - [Linux](#linux)
  - [Mac OS](#mac-os)
  - [Windows](#windows)
  - [Build from source](#build-from-source)
- [Privacy Notes](#privacy-notes)
- [Credits](#credits)

## Features

- Mails: Full client for Eve Mails
- Character stats: Display of current information about a character (e.g. location)

More features planned...

## Screenshot

![example](https://cdn.imgpile.com/f/dsxYy1a_xl.png)

## Installation

For installing this app on your desktop, please see the specific instructions for your platform. Alternatively you can also build the app directly from source.

### Linux

First download the linux tar file from the latest release on Github.

Then install the release on your desktop with:

```sh
sudo tar xvfJ evebuddy-v1.0.0-linux-amd64.tar.xz -C /
```

This will install the app for all users on your system. User specific data will be stored in the home directories of each user.

### Mac OS

First download the darwin zip file from the latest release on Github.

Then unzip the file into a directory of your choice and run the .app file to start the app.

### Windows

First download the windows zip file from the latest release on Github.

Then unzip the file into a directory of your choice and run the .exe file to start the app.

### Build from source

You can also build the app from source. For that your system needs to be able to build Fyne apps, which requires you to have installed the Go tools, a C compiler and a systems graphics driver. For details please see [Fyne - Getting started](https://docs.fyne.io/started/).

When you have all necessary tools installed, you can build and run this app direct from the repository with:

```sh
go run github.com/ErikKalkoken/evebuddy@latest
```

## Privacy notes

We understand and respect the privacy concerns of our fellow Eve Online players. Therefore all data of this app is stored and kept locally (e.g. character tokens and data). Internet requests by this app are made to CCPs game servers only.

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
