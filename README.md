# Merit Profile Renderer

[![MIT](https://img.shields.io/github/license/MieuxVoter/merit-profile-app?style=for-the-badge)](LICENSE)
[![Release](https://img.shields.io/github/v/release/MieuxVoter/merit-profile-app?include_prereleases&style=for-the-badge)](https://github.com/MieuxVoter/merit-profile-app/releases)
[![Discord Chat https://discord.gg/k9YRuZPSZs](https://img.shields.io/discord/705322981102190593.svg?style=for-the-badge)](https://discord.gg/k9YRuZPSZs)

A web app that renders merit profiles from tallies.

Try it out at https://educ.mieuxvoter.fr


## Features

- [x] Output pretty and readable SVG
- [x] Enter the tallies by hand using a form
- [x] Import a CSV file
- [x] Support static files (for favicon, CSS…)
- [x] Rank the proposals using Majority Judgment
- [ ] Balance tallies
- [x] Localization
- [x] Docker configuration for deployment


## Run locally

```shell
make build && build/mpa
```

or, using Docker:

```shell
make start
```

Then, visit http://localhost:8033

### Watch for changes and reload

During development, for fast iterations, it's handy to reload the webserver when a file changes.
I'm using `entr` for this, it works like a charm:

```bash
find src | entr -r go run src/main.go
```
