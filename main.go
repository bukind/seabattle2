package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"math/rand"
	"slices"

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

type Side int

type CellParams struct {
	SceneColor   color.RGBA
	CircleColor  color.RGBA
	CircleRadius float32
}

const (
	Ncells     = 8
	cellSize   = 32
	cellSizeF  = float32(cellSize)
	cellBorder = 1
	gameTPS    = 10
)

const (
	CellEmpty Cell = iota
	CellMiss
	CellMist // mist -- no ship
	CellHide // hidden ship
	CellShip
	CellFire
	CellDead
)

const (
	SideSelf Side = iota
	SidePeer
)

var (
	ptSansFontSource *text.GoTextFaceSource

	colorEmpty = color.RGBA{}
	colorMist  = color.RGBA{0xbb, 0xbb, 0xbb, 0xff}
	colorSea   = color.RGBA{0x00, 0x22, 0xff, 0xff}
	colorShip  = color.RGBA{0x22, 0x22, 0x22, 0xff}

	cellParams = map[Cell]CellParams{
		CellEmpty: {colorSea, colorEmpty, 0.},
		CellMiss:  {colorSea, colorMist, 0.25},
		CellMist:  {colorMist, colorEmpty, 0.},
		// TODO: change second color to empty and radius to 0.
		CellHide: {colorMist, colorShip, 0.1},
		CellShip: {colorShip, colorEmpty, 0.},
		CellFire: {color.RGBA{0x88, 0x22, 0x22, 0xff}, color.RGBA{0xff, 0x88, 0x22, 0x88}, 0.45},
		CellDead: {colorSea, colorShip, 0.55},
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
	case CellMiss:
		return "miss"
	case CellMist:
		return "mist"
	case CellHide:
		return "hide"
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

type XY struct {
	X, Y int
}

type Board struct {
	Game  *Game
	Side  Side
	Cells [][]Cell
}

func NewBoard(g *Game, side Side) *Board {
	rows := make([][]Cell, Ncells)
	cell := CellEmpty
	if side == SidePeer {
		cell = CellMist
	}
	for i := range rows {
		r := make([]Cell, Ncells)
		rows[i] = r
		for j := range r {
			r[j] = cell
		}
	}
	return &Board{
		Game:  g,
		Side:  side,
		Cells: rows,
	}
}

func (b *Board) draw(screen *ebiten.Image) {
	g := b.Game
	// Draw cells
	for x := 0; x < Ncells; x++ {
		for y := 0; y < Ncells; y++ {
			g.opts.GeoM.Reset()
			g.moveXY(&g.opts.GeoM, x, y, b.Side)
			b.drawCellInto(x, y, g.cellImage)
			screen.DrawImage(g.cellImage, &g.opts)
		}
		text.Draw(screen, fmt.Sprintf("%c", 'A'+x), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(x, Ncells, b.Side))
	}
}

func (b *Board) drawCellInto(x, y int, into *ebiten.Image) {
	g := b.Game
	c := b.Cells[y][x]
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

func (b *Board) addRandomShips(maxShipSize, retries int) error {
	num := 1
	for s := maxShipSize; s > 0; s-- {
		for n := 0; n < num; n++ {
			placed := false
			for attempt := 0; attempt < retries; attempt++ {
				if placed = b.placeShip(s); placed {
					break
				}
			}
			if !placed {
				return fmt.Errorf("cannot place ship of size %d", s)
			}
		}
		num++
	}
	return nil
}

func (b *Board) placeShip(s int) bool {
	dx := 1
	dy := s
	if r := rand.Intn(2); r != 0 {
		dx = s
		dy = 1
	}
	x0 := rand.Intn(Ncells - dx + 1)
	y0 := rand.Intn(Ncells - dy + 1)
	cell := CellShip
	if b.Side == SidePeer {
		cell = CellHide
	}
	if dx > 1 {
		if !(b.isCellsEmptyY(y0-1, x0, x0+dx) &&
			b.isCellsEmptyY(y0+1, x0, x0+dx) &&
			b.isCellsEmptyY(y0, x0-1, x0+dx+1)) {
			return false
		}
		for i := x0; i < x0+dx; i++ {
			b.Cells[y0][i] = cell
		}
	} else {
		if !(b.isCellsEmptyX(x0-1, y0, y0+dy) &&
			b.isCellsEmptyX(x0+1, y0, y0+dy) &&
			b.isCellsEmptyX(x0, y0-1, y0+dy+1)) {
			return false
		}
		for i := y0; i < y0+dy; i++ {
			b.Cells[i][x0] = cell
		}
	}
	return true
}

func (b *Board) isCellsEmptyX(x, y0, y1 int) bool {
	if x < 0 || x >= Ncells {
		return true
	}
	if y0 < 0 {
		y0 = 0
	}
	if y1 > Ncells {
		y1 = Ncells
	}
	for i := y0; i < y1; i++ {
		switch c := b.Cells[i][x]; c {
		case CellHide, CellShip, CellFire, CellDead:
			return false
		}
	}
	return true
}

func (b *Board) isCellsEmptyY(y, x0, x1 int) bool {
	if y < 0 || y >= Ncells {
		return true
	}
	if x0 < 0 {
		x0 = 0
	}
	if x1 > Ncells {
		x1 = Ncells
	}
	for i := x0; i < x1; i++ {
		switch c := b.Cells[y][i]; c {
		case CellHide, CellShip, CellFire, CellDead:
			return false
		}
	}
	return true
}

// return true if you can hit again.
func (b *Board) hitCell(x, y int) bool {
	switch c := b.Cells[y][x]; c {
	case CellEmpty, CellMist:
		b.Cells[y][x] = CellMiss
		return false
	case CellHide, CellShip:
		b.Cells[y][x] = CellFire
		if sunk := b.isShipSunk(x, y); len(sunk) > 0 {
			for _, xy := range sunk {
				b.Cells[xy.Y][xy.X] = CellDead
			}
		}
		return true
	}
	// In all other cases, we don't change the board state.
	// but ask to hit again.
	return true
}

// the ship is sunk when result is not empty, and it will contains all its cells.
func (b *Board) isShipSunk(x0, y0 int) []XY {
	result := make([]XY, 0, 4)
	for dx := -1; dx < 2; dx += 2 {
		for x := x0 + dx; x >= 0 && x < Ncells; x += dx {
			switch c := b.Cells[y0][x]; c {
			case CellShip, CellHide:
				return nil
			case CellFire:
				result = append(result, XY{x, y0})
			default:
				x = -10000
			}
		}
	}
	for dy := -1; dy < 2; dy += 2 {
		for y := y0 + dy; y >= 0 && y < Ncells; y += dy {
			switch c := b.Cells[y][x0]; c {
			case CellShip, CellHide:
				return nil
			case CellFire:
				result = append(result, XY{x0, y})
			default:
				y = -10000
			}
		}
	}
	return append(result, XY{x0, y0})
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
	cursor := g.CursorSelf
	side := SidePeer
	if g.WhoseTurn != SideSelf {
		cursor = g.CursorPeer
		side = SideSelf
	}
	g.moveXY(&g.opts.GeoM, cursor.X, cursor.Y, side)
	screen.DrawImage(g.cellImage, &g.opts)
}

type Game struct {
	Tick       int
	Boards     [2]*Board
	WhoseTurn  Side
	CursorSelf XY
	CursorPeer XY

	// cache objects.
	cellImage     *ebiten.Image
	vtx           []ebiten.Vertex
	idx           []uint16
	opts          ebiten.DrawImageOptions
	keys          []ebiten.Key
	activeTouches []ebiten.TouchID
	killedTouches []ebiten.TouchID
}

func NewGame() *Game {
	g := &Game{
		cellImage: ebiten.NewImage(cellSize, cellSize),
	}
	g.Boards = [2]*Board{NewBoard(g, SideSelf), NewBoard(g, SidePeer)}
	return g
}

func (g *Game) init() error {
	for _, b := range g.Boards {
		if err := b.addRandomShips(4, 30); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) Update() error {
	if g.Tick == 0 {
		if err := g.init(); err != nil {
			return err
		}
	}
	g.Tick++
	// log.Printf("update tick=%d", g.Tick)
	if err := g.handleKeys(); err != nil {
		return err
	}
	return g.handleTouches()
}

func (g *Game) handleKeys() error {
	g.keys = inpututil.AppendJustReleasedKeys(g.keys[:0])
	for _, k := range g.keys {
		if k == ebiten.KeyQ {
			// TODO: remove this.
			// Special handling even during peer turn.
			return ebiten.Termination
		}
		if g.WhoseTurn == SidePeer {
			continue
		}
		switch k {
		case ebiten.KeyArrowUp:
			g.CursorSelf.Y--
			if g.CursorSelf.Y < 0 {
				g.CursorSelf.Y = Ncells - 1
			}
		case ebiten.KeyArrowDown:
			g.CursorSelf.Y++
			if g.CursorSelf.Y >= Ncells {
				g.CursorSelf.Y = 0
			}
		case ebiten.KeyArrowLeft:
			g.CursorSelf.X--
			if g.CursorSelf.X < 0 {
				g.CursorSelf.X = Ncells - 1
			}
		case ebiten.KeyArrowRight:
			g.CursorSelf.X++
			if g.CursorSelf.X >= Ncells {
				g.CursorSelf.X = 0
			}
		case ebiten.KeySpace:
			if g.Boards[SidePeer].hitCell(g.CursorSelf.X, g.CursorSelf.Y) {
				// TODO: hit, check that the game is won.
			} else {
				g.WhoseTurn = SidePeer
			}
		}
	}
	return nil
}

func (g *Game) handleTouches() error {
	g.activeTouches = inpututil.AppendJustPressedTouchIDs(g.activeTouches)
	slices.Sort(g.activeTouches)
	g.killedTouches = inpututil.AppendJustReleasedTouchIDs(g.killedTouches[:0])
	slices.Sort(g.killedTouches)
	// Remove killedTouches from activeTouches
	i, j := 0, 0
	for k := 0; k < len(g.killedTouches) && i < len(g.activeTouches); i++ {
		if g.activeTouches[i] == g.killedTouches[k] {
			// if the touch is released.
			k++
		} else {
			// the touch is not released yet.
			g.activeTouches[j] = g.activeTouches[i]
			j++
		}
	}
	g.activeTouches = append(g.activeTouches[:j], g.activeTouches[i:]...)
	if g.WhoseTurn != SideSelf {
		return nil
	}
	// Draw cursor at the active touches.
	for _, t := range g.activeTouches {
		// TODO: if touch is outside the board, do not hit it.
		tx, ty := ebiten.TouchPosition(t)
		log.Printf("touch (%d, %d)", tx, ty)
		g.CursorSelf.X = pos2Cell(tx, Ncells+1, Ncells)
		g.CursorSelf.Y = pos2Cell(ty, 1, Ncells)
	}
	for _, t := range g.killedTouches {
		// TODO: if touch is outside the board, do not hit it.
		tx, ty := inpututil.TouchPositionInPreviousTick(t)
		x := pos2Cell(tx, Ncells+1, Ncells)
		y := pos2Cell(ty, 1, Ncells)
		if g.Boards[SidePeer].hitCell(x, y) {
			// TODO: hit, check that the game is won.
		} else {
			g.WhoseTurn = SidePeer
			return nil
		}
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
func (g *Game) moveXY(m *ebiten.GeoM, x, y int, side Side) {
	m.Translate(float64(cellPos(int(side)*(Ncells+1)+x)), float64(cellPos(y+1)))
}

// textInXY returns text options for the text centered in cell (X,Y)
func (g *Game) textInXY(x, y int, side Side) *text.DrawOptions {
	topts := &text.DrawOptions{}
	topts.PrimaryAlign = text.AlignCenter
	topts.SecondaryAlign = text.AlignCenter
	topts.GeoM.Translate(cellSize*0.5, cellSize*0.5)
	g.moveXY(&topts.GeoM, x, y, side)
	return topts
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.Boards[0].draw(screen)
	g.Boards[1].draw(screen)
	// Draw vertical numbers between boards.
	for y := 0; y < Ncells; y++ {
		text.Draw(screen, fmt.Sprintf("%c", '1'+y), &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(Ncells, y, SideSelf))
	}
	g.drawCursor(screen)
}

func (g *Game) Layout(oW, oH int) (int, int) {
	return cellPos(Ncells*2 + 1), cellPos(Ncells + 2)
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
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
