# oncall-status

A PagerDuty utility app to show if you're oncall. Also shows a notification when a new incident has been triggered.

![Go](https://github.com/mtougeron/oncall-status/workflows/Go/badge.svg) ![Gosec](https://github.com/mtougeron/oncall-status/workflows/Gosec/badge.svg) [![GitHub release](https://img.shields.io/github/v/release/mtougeron/oncall-status?sort=semver)](https://github.com/mtougeron/oncall-status/releases)

## Installation

From the [releases page](https://github.com/mtougeron/oncall-status/releases) download the latest DMG file. Once downloaded, drag the `OncallStatus.app` to the `Applications` folder.

## Basic build & test

```
make build && cp -a OncallStatus.app /Applications
```

## Build & sign a release

The signing & notarizing with Apple happen using [`gon`](https://github.com/mitchellh/gon).

### Automation

This will create a release and upload the sign & notarized application to the release.

```
git tag -a v#.#.# -m "<release notes>"
git push origin v#.#.#
```

### Local testing

To test locally you can run the following `make` commands

`make build` -- Build the application binary

`make sign-app` -- Sign the application binary. Must set `AC_USERNAME` & `AC_PASSWORD` environment variables to work.
If you are testing outside of the main application the `bundle_id` inside of `gon-app.hcl` will need to be changed as well.

`make dmg` -- Create the DMG for the application

`make notarize-dmg` -- Notarize the DMG with Apple. Must set `AC_USERNAME` & `AC_PASSWORD` environment variables to work.
If you are testing outside of the main application the `bundle_id` inside of `gon-dmg.hcl` will need to be changed as well.
