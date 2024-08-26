package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/bukind/seabattle2/fonts"
)

const (
	cellSize   = 32
	cellBorder = 1
)

var (
	ptSansFontSource *text.GoTextFaceSource
)

type Cell struct {
	Color color.Color
}

type Row []Cell

type Board struct {
	Rows []Row
}

func NewBoard(size int) *Board {
	rows := make([]Row, size)
	for i := range rows {
		r := make(Row, size)
		rows[i] = r
		for j := 0; j < size; j++ {
			r[j].Color = color.RGBA{uint8(0xff * j / (size - 1)), uint8(0xff * i / (size - 1)), 0x78, 0xff}
		}
	}
	return &Board{
		Rows: rows,
	}
}

type Game struct {
	Size      int
	Boards    [2]*Board
	cellImage *ebiten.Image
}

func NewGame(size int) *Game {
	g := &Game{
		Size:      size,
		Boards:    [2]*Board{NewBoard(size), NewBoard(size)},
		cellImage: ebiten.NewImage(cellSize, cellSize),
	}
	return g
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	topts := &text.DrawOptions{}
	for i := 0; i < g.Size; i++ {
		for b := 0; b < 2; b++ {
			for j := 0; j < g.Size; j++ {
				opts.GeoM.Reset()
				opts.GeoM.Translate(float64((b*(g.Size+1)+j)*(cellSize+cellBorder)), float64(cellSize+i*(cellSize+cellBorder)))
				g.cellImage.Fill(g.Boards[b].Rows[i][j].Color)
				screen.DrawImage(g.cellImage, opts)
			}
			topts.GeoM.Reset()
			topts.GeoM.Translate(float64((b*(g.Size+1)+i)*(cellSize+cellBorder)), float64(cellSize+g.Size*(cellSize+cellBorder)))
			text.Draw(screen, fmt.Sprintf("%c", 'A'+i), &text.GoTextFace{
				Source: ptSansFontSource,
				Size: cellSize,
			}, topts)
		}
		topts.GeoM.Reset()
		topts.GeoM.Translate(float64((g.Size)*(cellSize+cellBorder)), float64((g.Size-i)*(cellSize+cellBorder)))
		text.Draw(screen, fmt.Sprintf("%c", '1'+i), &text.GoTextFace{
			Source: ptSansFontSource,
			Size: cellSize,
		}, topts)
	}
}

func (g *Game) Layout(oW, oH int) (int, int) {
	size := g.Size*(cellSize+cellBorder) + cellBorder
	return size*2 + cellSize, size+2*cellSize
}

func loadFonts() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.PTSansRegular))
	if err != nil {
		log.Fatal(err)
	}
	ptSansFontSource = s
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	loadFonts()
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("sea battle")
	ebiten.SetTPS(1)
	cells := 8
	if err := ebiten.RunGame(NewGame(cells)); err != nil {
		log.Fatal(err)
	}
}
