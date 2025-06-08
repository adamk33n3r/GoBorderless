![GoBorderless Icon](/res/icon128.png)

# GoBorderless

GoBorderless is a modern, open-source Windows application that allows users to easily manage and toggle borderless windowed mode for any application (especially useful for games that have poor alt-tabbing support). Initially created as a clone of [NoMoreBorder](https://github.com/invcble/NoMoreBorder) for the purpose of learning Go, GoBorderless now offers a clean UI, streamlined user experience, and additional features.

## Features

- 🪟 **Make Any Window Borderless**  
  Instantly remove window borders from any running application with a single click.

- 🔍 **Automatic Window Detection**  
  Automatically lists currently open windows with their titles and executable paths.

- 🖱️ **One-Click Control**  
  Select a window from the list and toggle borderless mode effortlessly.

- 🧠 **Smart Positioning**  
  Remembers your last settings and positions for each app.

- ⚙️ **Lightweight & Portable**  
  No installer needed. Just download and run the portable executable.

- 🛠️ **Written in Go with Fyne**  
  Clean, fast, and cross-platform capable core built using the [Fyne](https://fyne.io/) GUI toolkit.

## Screenshots

![App](/screenshot-app.png)
![Config](/screenshot-config.png)

## Download

You can find the latest portable `.exe` files in the [releases](https://github.com/adamk33n3r/GoBorderless/releases) section.

## Build from Source

To build GoBorderless yourself:

```sh
go install fyne.io/tools/cmd/fyne@latest # currently need https://github.com/adamk33n3r/fyne-tools for correct icon support unless merged upstream
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

Otherwise a simple `go run .` should work.
