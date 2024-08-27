package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"

	"github.com/bukind/seabattle2/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// bbbbbbbbbbbbbbbbbb
// bcccccccccbccc
// bcccccccccbccc

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
	Ncells    int
	Boards    [2]*Board
	cellImage *ebiten.Image
}

func NewGame(nCells int) *Game {
	g := &Game{
		Ncells:    nCells,
		Boards:    [2]*Board{NewBoard(nCells), NewBoard(nCells)},
		cellImage: ebiten.NewImage(cellSize, cellSize),
	}
	return g
}

func (g *Game) Update() error {
	return nil
}

func cellPos(row int) int {
	return cellBorder + (cellSize+cellBorder)*row
}

// moveXY translates GeoM into the cell (X,Y) coordinate of the cell on the board.
func (g *Game) moveXY(m *ebiten.GeoM, col, row, board int) {
	m.Translate(float64(cellPos(board*(g.Ncells+1)+col)), float64(cellPos(row+1)))
}

func (g *Game) Draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	topts := func(x, y, b int) *text.DrawOptions {
		topts := &text.DrawOptions{}
		topts.PrimaryAlign = text.AlignCenter
		topts.SecondaryAlign = text.AlignCenter
		topts.GeoM.Translate(cellSize*0.5, cellSize*0.5)
		g.moveXY(&topts.GeoM, x, y, b)
		return topts
	}
	for y := 0; y < g.Ncells; y++ {
		for b := 0; b < 2; b++ {
			for x := 0; x < g.Ncells; x++ {
				opts.GeoM.Reset()
				g.moveXY(&opts.GeoM, x, y, b)
				g.cellImage.Fill(g.Boards[b].Rows[y][x].Color)
				screen.DrawImage(g.cellImage, opts)
			}
			text.Draw(screen, fmt.Sprintf("%c", 'A'+y), &text.GoTextFace{
				Source: ptSansFontSource,
				Size:   cellSize * 0.8,
			}, topts(y, g.Ncells, b))
		}
		text.Draw(screen, fmt.Sprintf("%c", '1'+y), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, topts(g.Ncells, g.Ncells-y-1, 0))
	}
}

func (g *Game) Layout(oW, oH int) (int, int) {
	return cellPos(g.Ncells*2 + 1), cellPos(g.Ncells + 2)
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
	nCells := 8
	if err := ebiten.RunGame(NewGame(nCells)); err != nil {
		log.Fatal(err)
	}
}
