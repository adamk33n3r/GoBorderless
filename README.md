# GoBorderless
What started off as a clone of [NoMoreBorder](https://github.com/invcble/NoMoreBorder) I used to learn Go, GoBorderless is a modern and easy-to-use borderless window manager for Windows. It's currently in feature-parity with NoMoreBorder, but I'm working on adding more features and improving the UI/UX.

You can find a portable exe in the [releases](https://github.com/adamk33n3r/GoBorderless/releases) section.
Or if you'd like to build it yourself, you can clone this repository and run the following in the root directory:
```sh
go install fyne.io/tools/cmd/fyne@latest
fyne package --release
```

## Development

GoBorderless is written in Go using the [Fyne](https://fyne.io/) framework.

I use [air](https://github.com/air-verse/air) to live-reload the app as I make changes. You can install it using the following command:
```sh
go install github.com/air-verse/air@latest
```
And then just run
```sh
air
```
to start it. It uses the `.air.toml` file in the repository by default.

TODO:
Multiple of same app (prevent? or prevent auto-apply to all?)
Validation
