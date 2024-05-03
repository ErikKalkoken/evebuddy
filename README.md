# Eve Buddy

Eve Buddy is a companion app for [Eve Online](https://www.eveonline.com/) players. It runs on Windows, MacOS and LINUX desktops.

![build status](https://github.com/ErikKalkoken/evebuddy/actions/workflows/go.yml/badge.svg)

This app is in development.

## Features

- Mails: Receiving, sending and deleting of Eve mails
- Character stats: Display of current information about a character (e.g. location)

More features planned...

## Installation

### Linux

First download the latest release from Github.

Then install the release on your desktop with:

```sh
sudo tar xvfJ evebuddy_linux_amd64.tar.xz -C /
```

When everything worked correctly the app should appear under "Applications".

### Windows

tbd

### Mac OS

tbd

## Privacy notes

We understand and respect the privacy concerns of our fellow Eve Online players. Therefore all data of this app is stored locally (e.g. character tokens) and not shared with any 3rd party. Internet requests by this app are made to CCP's game server only.

## Developer notes

In general this app can be developed on many popular desktop platforms including Windows, MacOS and Linux.

To setup a local development environment please follow the platform specific instructions for setting up development for a [fye app](https://docs.fyne.io/started/). Fyne is the GUI framework used by Eve Buddy.

## Credits

"EVE", "EVE Online", "CCP", and all related logos and images are trademarks or registered trademarks of CCP hf.
