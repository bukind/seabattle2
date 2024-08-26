package main

import (
	"image/color"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	cellSize   = 32
	cellBorder = 1
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
	for b := 0; b < 2; b++ {
		for i := 0; i < g.Size; i++ {
			for j := 0; j < g.Size; j++ {
				opts.GeoM.Reset()
				opts.GeoM.Translate(float64((b*(g.Size+1)+j)*(cellSize+cellBorder)), float64(cellSize+i*(cellSize+cellBorder)))
				g.cellImage.Fill(g.Boards[b].Rows[i][j].Color)
				screen.DrawImage(g.cellImage, opts)
			}
		}
	}
}

func (g *Game) Layout(oW, oH int) (int, int) {
	size := g.Size*(cellSize+cellBorder) + cellBorder
	return size*2 + cellSize, size+2*cellSize
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("sea battle")
	ebiten.SetTPS(1)
	cells := 8
	if err := ebiten.RunGame(NewGame(cells)); err != nil {
		log.Fatal(err)
	}
}
