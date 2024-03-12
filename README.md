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
- [ ] Show mailing lists as folder
- [ ] Update "is_read" for mails from ESI
- [ ] Add error handling for mailing lists
- [ ] Add basic unit tests

### Future releases

- Ability to sent new mail
- Ability to update "is_read" from client
- Add additional functionality, e.g. contacts (aka address book)
- Full unit tests
- Packaging for Linux, Windows and MAC
- Store logs in a file instead of console
- Add settings
- Show mail bodies as rich text
