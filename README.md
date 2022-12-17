# Introduction

This is a reworking of <https://github.com/orivej/static-appimage> that uses squashfs instead of zip.

## Usage

```sh
go get github.com/orivej/static-appimage/...
make-static-appimage APPDIR DESTINATION
```

`APPDIR` must already contain an `AppRun`.
