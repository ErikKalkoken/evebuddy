---
title: Install
lead: How to install, update and uninstall EVE Buddy on Windows, macOS, Linux and Android.
description: Installation, update and uninstall guides for EVE Buddy.
toc: true
---

To install EVE Buddy, [download](download/) the latest release for your platform. The app ships as a single executable file that can be run directly. When you run the app for the first time it will automatically install itself for the current user (i.e. by creating folders in the home folder of the current user).

## Windows

1. Download the Windows zip file.
1. Unzip the file into a directory of your choice and run the .exe file to start the app.

> [!TIP]
> Windows Defender (and similar programs) may incorrectly report EVE Buddy as containing a trojan. This is usually a false positive caused by a [known issue](https://github.com/microsoft/go/issues/1255) with programs made with the Go programming language. Each EVE Buddy release is built from scratch on a fresh Windows container provided by GitHub, so it is highly unlikely to be infected. If this happens to you, please exclude EVE Buddy's executable from Windows Defender (and similar programs) to proceed.

## macOS

1. Download the macOS zip file for your Mac (Apple Silicon or Intel).
1. Unzip the file into a directory of your choice.
1. Run the .app file to start the app.

> [!TIP]
> macOS may incorrectly report EVE Buddy as "damaged", because it is not signed with an Apple certificate. You can remove this error by opening a terminal and running the following command. For more information please see [Fyne Troubleshooting](https://docs.fyne.io/faq/troubleshoot#distribution):
>
> ```sh
> sudo xattr -r -d com.apple.quarantine "EVE Buddy.app"
> ```

## Linux

We provide two variants for installing on Linux desktop:

- **Tarball**: Requires installation, but also integrates the app into your desktop environment. The tarball also has wider compatibility among different Linux versions.
- **AppImage**: Run the app directly from the executable without requiring installation or root access.

### Tarball

1. Download the tar file.
1. Decompress the tar file, for example with: `tar xf evebuddy-0.33.0-linux-amd64.tar.xz`
1. Change into the extracted folder: `cd evebuddy`
1. Run `make user-install` to install the app for the current user or run `sudo make install` to install the app on the system.

You should now have a shortcut in your desktop environment's launcher for starting the app.

### AppImage

The [AppImage](https://appimage.org/) format can be used without installation and runs on many different Linux distributions.

1. Download the AppImage file.
1. Make the AppImage file executable.
1. Execute the AppImage file to start the app.

> [!TIP]
> Should you get the error `AppImages require FUSE to run.`, you need to install FUSE on your system first. This library is required by all AppImages. Please see [this page](https://docs.appimage.org/user-guide/troubleshooting/fuse.html#the-appimage-tells-me-it-needs-fuse-to-run) for details.

## Android

> [!NOTE]
> The app can currently only be installed from our website or GitHub and is not available in any of the Android app stores.

> [!IMPORTANT]
> EVE Buddy needs unrestricted background usage in order to function properly. Otherwise Android automatically suspends apps when you switch to another app. Then you can no longer add new characters, because this requires you to switch to your browser app while EVE Buddy keeps running. EVE Buddy also needs to keep running in order to pick up events for notifications.
>
> Please also make sure you do not have power saving mode enabled (e.g. on Samsung Galaxy), which would also restrict background app usage.

Here is how to install EVE Buddy on your Android phone:

1. Make sure you have enabled in settings that you can install apps from your web browser (for detailed instructions please see [How to install Unknown Sources applications in Android](https://www.appaloosa.io/blog/guides/how-to-install-apps-from-unknown-sources-in-android)).
1. Open the [download page](download/) in your mobile browser.
1. Download the Android APK file (this can take a minute).
1. When prompted, choose to install the file / open the file with the default installer.
1. In case you get a security warning from Google Play Protect:
   1. Select "More details"
   1. Select "Install anyway"
1. Enable unrestricted background usage for EVE Buddy in the Android Settings app (see [below](#enable-unrestricted-background-usage)).
1. Optional: Enable notifications for EVE Buddy (see [below](#enable-notifications)).

### Enable unrestricted background usage

1. Open the Settings app.
1. Go to Apps and choose EVE Buddy.
1. Choose "App battery usage".
1. Select "Allow background usage".
1. Make sure the "Unrestricted" option is selected. That's it.

![App info for EVE Buddy](images/android-settings-1.png)
![App battery usage](images/android-settings-2.png)
![Allow background usage set to Unrestricted](images/android-settings-3.png)
{.screens}

### Enable notifications

1. In EVE Buddy, go to Settings and enable which kinds of notifications you want to receive (e.g. mails).
1. Send a test notification (via the menu in Settings).
1. When asked, allow EVE Buddy to send Android notifications.

You should now see a test notification.

## Updating

The app will inform you when a new version is available. To update, [download](download/) and install the newest version for your platform.

## Uninstalling

### Windows, Linux and macOS

The desktop versions have a special mode for removing user data. First, run the app with the following option:

```sh
./evebuddy --delete-data
```

This will ask for confirmation and then delete your user data like characters, log files, etc. from your computer.

Then delete the executable itself:

```sh
rm evebuddy
```

If you installed the tarball, run either `make user-uninstall` or `sudo make uninstall` instead, depending on how you installed it.

### Android

On Android you can uninstall the app via the system's Settings app. This will also remove all data.
