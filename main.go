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

type Cell color.Color

type Row []Cell

type Board struct {
	Game *Game
	Side int
	Rows []Row
}

func NewBoard(g *Game, side, size int) *Board {
	rows := make([]Row, size)
	for i := range rows {
		r := make(Row, size)
		rows[i] = r
		for j := 0; j < size; j++ {
			r[j] = color.RGBA{uint8(0xff * j / (size - 1)), uint8(0xff * i / (size - 1)), 0x78, 0xff}
		}
	}
	return &Board{
		Game: g,
		Side: side,
		Rows: rows,
	}
}

func (b *Board) draw(screen *ebiten.Image) {
	opts := &ebiten.DrawImageOptions{}
	g := b.Game
	// Draw cells
	for x := 0; x < g.Ncells; x++ {
		for y := 0; y < g.Ncells; y++ {
			opts.GeoM.Reset()
			g.moveXY(&opts.GeoM, x, y, b.Side)
			g.cellImage.Fill(b.Rows[y][x])
			screen.DrawImage(g.cellImage, opts)
		}
		text.Draw(screen, fmt.Sprintf("%c", 'A'+x), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(x, g.Ncells, b.Side))
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
		cellImage: ebiten.NewImage(cellSize, cellSize),
	}
	g.Boards = [2]*Board{NewBoard(g, 0, nCells), NewBoard(g, 1, nCells)}
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

// textInXY returns text options for the text centered in cell (X,Y)
func (g *Game) textInXY(col, row, board int) *text.DrawOptions {
	topts := &text.DrawOptions{}
	topts.PrimaryAlign = text.AlignCenter
	topts.SecondaryAlign = text.AlignCenter
	topts.GeoM.Translate(cellSize*0.5, cellSize*0.5)
	g.moveXY(&topts.GeoM, col, row, board)
	return topts
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.Boards[0].draw(screen)
	g.Boards[1].draw(screen)
	// Draw vertical numbers between boards.
	for y := 0; y < g.Ncells; y++ {
		text.Draw(screen, fmt.Sprintf("%c", '1'+y), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(g.Ncells, g.Ncells-y-1, 0))
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
