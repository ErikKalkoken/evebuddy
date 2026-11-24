---
# Text for the home page. The layout is in layouts/home.html.
title: EVE Buddy

hero:
  title: Your companion for EVE Online
  lead: Keep track of all your characters and corporations, utilize consolidated views and get notified about what matters – on your desktop and your Android phone.
  badges:
    - Free & open source
    - Your data stays on your device
    - Windows · Linux · macOS  · Android
  # Image paths are relative to static/.
  images:
    desktop:
      src: images/desktop.png
      alt: EVE Buddy desktop version
    phone:
      src: images/mobile.webp
      alt: EVE Buddy mobile version
  featured:
    text: Featured in EVE Online's official Community Beat
    links:
      - label: August 2025
        url: https://www.eveonline.com/news/view/community-beat-for-16-august-2025
      - label: November 2024
        url: https://www.eveonline.com/news/view/community-beat-for-22-november

highlights:
  title: All your characters. One app.
  text: EVE Buddy brings the information you need from New Eden to your desktop and phone.
  # Image paths are relative to static/. Image sides alternate automatically.
  items:
    - eyebrow: Consolidated views
      title: All your characters and corporations in one view
      text: "Get answers the game can't give you at a glance: across every character and corporation."
      image:
        src: images/overview_highlight.png
        alt: Asset Search on desktop
      items:
        - Search assets across all characters and export them
        - See which character has industry slots available
        - Forecast when each PI colony stops working
        - Find the nearest jump clone and track wealth
    - eyebrow: Characters
      title: Keep track of each character
      text: Everything about a character in one place, without logging in to the game.
      image:
        src: images/character_highlight.png
        alt: Character monitor on desktop
      items:
        - Training queue and trained skills, with export for PyFA
        - Assets at all locations
        - Clones, implants and jump cooldown
        - Wallet, market transactions and loyalty points
    - eyebrow: Corporations
      title: Keep your corporation running
      text: Stay on top of your corporation's structures, finances and industry.
      image:
        src: images/corporation_highlight.png
        alt: Corporation monitor on desktop
      items:
        - Structures with fuel status, state and timers
        - Wallets and market transactions
        - Industry jobs and corporation assets
        - Members and wealth

# Icons: see layouts/_partials/icon.html for available names.
cards:
  title: More key features
  items:
    - icon: bell
      title: Notifications
      text: Get notified about structure attacks, new mail, empty training queues, contract changes and stopped PI colonies.
    - icon: search
      title: New Eden search
      text: Search live on the game server for characters, corporations, systems, types and more.
    - icon: info
      title: Information windows
      text: Details for most objects on screen, e.g. the sender of a mail, just like the in-game information windows.
    - icon: mail
      title: Mail client
      text: Send and receive EVE mail for all your characters.
    - icon: background
      title: Run in background
      text: Keeps running and notifying you while you do something else, like playing EVE. Minimizes to the system tray on desktop.
    - icon: palette
      title: Customizable UI
      text: Dark or light theme, and custom UI scaling on desktop.

---
