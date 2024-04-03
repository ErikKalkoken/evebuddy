# esiapp

A desktop client for Eve mails.

## To-Dos

### Minimal viable product

MVP is reaching feature parity with Member Audit mail.

- [x] Fetch mail bodies concurrently
- [x] Add paging to allow fetching more then 50 mails from ESI
- [x] Add error handling when fetching mails concurrently
- [x] Preselect mail after opening app
- [x] Auto retry ESI calls on known common errors
- [x] Allow selecting mails by folder
- [x] Refactor gui part into own package
- [x] Update list of characters after adding new character
- [x] Show mailing lists as folder
- [x] Update "is_read" for mails from ESI
- [ ] Calculate unread counts for the folders from local mails
- [ ] Add error handling for mailing lists
- [ ] Add new mails "on the fly" when refreshing
- [ ] Add basic unit tests
- [ ] Store logs in a file instead of console

### Future releases

- Add additional functionality, e.g. contacts (aka address book)
- Full unit tests
- Packaging for Linux, Windows and MAC
- Add settings
- Show mail bodies as rich text
