package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"

	"github.com/bukind/seabattle2/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// bbbbbbbbbbbbbbbbbb
// bcccccccccbccc
// bcccccccccbccc

type Cell int

type CellParams struct {
	SceneColor   color.RGBA
	CircleColor  color.RGBA
	CircleRadius float32
}

const (
	cellSize   = 32
	cellSizeF  = float32(cellSize)
	cellBorder = 1
	gameTPS    = 10

	CellEmpty Cell = iota
	CellHide
	CellMiss
	CellShip
	CellFire
	CellDead
)

var (
	ptSansFontSource *text.GoTextFaceSource

	colorEmpty = color.RGBA{}
	colorMist  = color.RGBA{0xbb, 0xbb, 0xbb, 0xff}
	colorSea   = color.RGBA{0x00, 0x22, 0xff, 0xff}
	colorShip  = color.RGBA{0x22, 0x22, 0x22, 0xff}

	cellParams = map[Cell]CellParams{
		CellEmpty: {colorSea, colorEmpty, 0.},
		CellHide:  {colorMist, colorEmpty, 0.},
		CellMiss:  {colorSea, colorMist, 0.25},
		CellShip:  {colorShip, colorEmpty, 0.},
		CellFire:  {color.RGBA{0x88, 0x22, 0x22, 0xff}, color.RGBA{0xff, 0x88, 0x22, 0x88}, 0.45},
		CellDead:  {colorSea, colorShip, 0.55},
	}

	fillImage = func() *ebiten.Image {
		img := ebiten.NewImageWithOptions(image.Rect(-1, -1, 1, 1), nil)
		img.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
		return img
	}()
)

func (c Cell) String() string {
	switch c {
	case CellEmpty:
		return "empty"
	case CellHide:
		return "hide"
	case CellMiss:
		return "miss"
	case CellShip:
		return "ship"
	case CellFire:
		return "fire"
	case CellDead:
		return "dead"
	}
	return "unknown"
}

func colorS(c color.RGBA) string {
	if c == colorEmpty {
		return "#none"
	}
	return fmt.Sprintf("#%x%x%x%x", c.R, c.G, c.B, c.A)
}

func setVtxColor(v *ebiten.Vertex, c color.RGBA) {
	v.ColorR = float32(c.R) / 0xff
	v.ColorG = float32(c.G) / 0xff
	v.ColorB = float32(c.B) / 0xff
	v.ColorA = float32(c.A) / 0xff
}

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
			r[j] = Cell(rand.Intn(int(CellDead + 1)))
		}
	}
	return &Board{
		Game: g,
		Side: side,
		Rows: rows,
	}
}

func (b *Board) draw(screen *ebiten.Image) {
	g := b.Game
	// Draw cells
	for x := 0; x < g.Ncells; x++ {
		for y := 0; y < g.Ncells; y++ {
			g.opts.GeoM.Reset()
			g.moveXY(&g.opts.GeoM, x, y, b.Side)
			b.drawCellInto(x, y, g.cellImage)
			screen.DrawImage(g.cellImage, &g.opts)
		}
		text.Draw(screen, fmt.Sprintf("%c", 'A'+x), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(x, g.Ncells, b.Side))
	}
}

func (b *Board) drawCellInto(x, y int, into *ebiten.Image) {
	g := b.Game
	c := b.Rows[y][x]
	params := cellParams[c]
	if params.SceneColor != colorEmpty {
		into.Fill(params.SceneColor)
	} else {
		into.Fill(color.RGBA{0, 0, 0, 0})
	}
	if params.CircleColor != colorEmpty {
		// Draw a circle in the middle of the cell.
		var path vector.Path
		path.Arc(cellSizeF/2, cellSizeF/2, cellSize*params.CircleRadius, 0, math.Pi*2, vector.Clockwise)
		path.Close()
		g.vtx, g.idx = path.AppendVerticesAndIndicesForFilling(g.vtx[:0], g.idx[:0])
		for i := range g.vtx {
			setVtxColor(&g.vtx[i], params.CircleColor)
		}
		op := &ebiten.DrawTrianglesOptions{
			AntiAlias: true,
			FillRule:  ebiten.FillRuleNonZero,
		}
		into.DrawTriangles(g.vtx, g.idx, fillImage, op)
	}
}

func (g *Game) drawCursor(screen *ebiten.Image) {
	col := color.RGBA{0, 0xff, 0, uint8(0xff * (g.Tick % (gameTPS + 1)) / gameTPS)}

	var path vector.Path
	hw := cellSizeF / 2
	dw := cellSizeF * 0.45
	path.MoveTo(hw-dw, hw-dw)
	path.LineTo(hw+dw, hw-dw)
	path.LineTo(hw+dw, hw+dw)
	path.LineTo(hw-dw, hw+dw)
	path.Close()
	g.vtx, g.idx = path.AppendVerticesAndIndicesForStroke(g.vtx[:0], g.idx[:0], &vector.StrokeOptions{
		Width:    cellSize / 8,
		LineJoin: vector.LineJoinRound,
	})
	for i := range g.vtx {
		setVtxColor(&g.vtx[i], col)
	}
	g.cellImage.Fill(color.RGBA{0, 0, 0, 0})
	g.cellImage.DrawTriangles(g.vtx, g.idx, fillImage, &ebiten.DrawTrianglesOptions{
		AntiAlias: true,
		FillRule:  ebiten.FillRuleNonZero,
	})
	g.opts.GeoM.Reset()
	g.moveXY(&g.opts.GeoM, g.CursorX, g.CursorY, 1)
	screen.DrawImage(g.cellImage, &g.opts)
}

type Game struct {
	Tick    int
	Ncells  int
	Boards  [2]*Board
	CursorX int
	CursorY int

	// cache objects.
	cellImage *ebiten.Image
	vtx       []ebiten.Vertex
	idx       []uint16
	opts      ebiten.DrawImageOptions
	keys      []ebiten.Key
	touchs    []ebiten.TouchID
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
	g.Tick++
	// log.Printf("update tick=%d", g.Tick)
	g.keys = inpututil.AppendJustReleasedKeys(g.keys[:0])
	for _, k := range g.keys {
		switch k {
		case ebiten.KeyArrowUp:
			g.CursorY--
			if g.CursorY < 0 {
				g.CursorY = g.Ncells - 1
			}
		case ebiten.KeyArrowDown:
			g.CursorY++
			if g.CursorY >= g.Ncells {
				g.CursorY = 0
			}
		case ebiten.KeyArrowLeft:
			g.CursorX--
			if g.CursorX < 0 {
				g.CursorX = g.Ncells - 1
			}
		case ebiten.KeyArrowRight:
			g.CursorX++
			if g.CursorX >= g.Ncells {
				g.CursorX = 0
			}
		case ebiten.KeySpace:
			g.Boards[1].Rows[g.CursorY][g.CursorX] = CellMiss
		}
	}
	g.touchs = inpututil.AppendJustReleasedTouchIDs(g.touchs[:0])
	for _, t := range g.touchs {
		// TODO: if touch is outside the board, do not hit it.
		tx, ty := inpututil.TouchPositionInPreviousTick(t)
		log.Printf("touch (%d, %d)", tx, ty)
		g.CursorX = pos2Cell(tx, g.Ncells+1, g.Ncells)
		g.CursorY = pos2Cell(ty, 1, g.Ncells)
		g.Boards[1].Rows[g.CursorY][g.CursorX] = CellMiss
	}
	return nil
}

func cellPos(row int) int {
	return cellBorder + (cellSize+cellBorder)*row
}

func pos2Cell(pos int, min, max int) int {
	c := (pos-cellBorder)/(cellSize+cellBorder) - min
	if c < 0 {
		c = 0
	}
	if c >= max {
		c = max - 1
	}
	return c
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
		}, g.textInXY(g.Ncells, y, 0))
	}
	g.drawCursor(screen)
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
	ebiten.SetTPS(gameTPS)
	nCells := 8
	if err := ebiten.RunGame(NewGame(nCells)); err != nil {
		log.Fatal(err)
	}
}
