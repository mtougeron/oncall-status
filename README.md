# oncall-status

A PagerDuty utility app to show if you're oncall. Also shows a notification when a new incident has been triggered.

## Basic build & test

```
make build && cp -a OncallStatus.app /Applications
```

## Build & sign a release

This will eventually be moved to a GH action

```
export AC_USERNAME="the apple connect username"
export AC_PASSWORD="the password for the username"
make build
make sign
```

Upload the `oncall-status.dmg` to the GitHub release
