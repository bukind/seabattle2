module github.com/bukind/seabattle2

go 1.22.0

toolchain go1.22.9

require github.com/hajimehoshi/ebiten/v2 v2.8.6

replace (
	github.com/bukind/seabattle2 => ../seabattle2
	github.com/bukind/seabattle2/fonts => ../seabattle2/fonts
)

require (
	github.com/ebitengine/gomobile v0.0.0-20240911145611-4856209ac325 // indirect
	github.com/ebitengine/hideconsole v1.0.0 // indirect
	github.com/ebitengine/purego v0.8.0 // indirect
	github.com/go-text/typesetting v0.2.0 // indirect
	github.com/jezek/xgb v1.1.1 // indirect
	golang.org/x/image v0.20.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)
