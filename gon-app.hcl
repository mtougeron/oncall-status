source = ["./OncallStatus.app", "./OncallStatus.app/Contents/OncallStatus"]
bundle_id = "com.github.mtougeron.oncall-status"

apple_id {
  password = "@env:AC_PASSWORD"
}

sign {
  application_identity = "63B65F5D57B165EE22DE1DACA8A474A6E7C5564E"
}
