package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"log"
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
	Ncells          = 8
	cellSize        = 32
	cellSizeF       = float32(cellSize)
	cellBorder      = 1
	gameTPS         = 20
	peerTicksPerAct = gameTPS / 10
	maxShipSize     = 4
)

const (
	CellEmpty Cell = iota
	CellMiss       // empty cell being hit
	CellMist       // mist -- empty cell hidden by the mist
	CellHide       // hidden ship
	CellShip       // ship cell
	CellFire       // ship on fire
	CellSunk       // sunk ship
	CellOily       // hidden mark around sunk ship, also are used for placement
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
	colorShip  = color.RGBA{0x44, 0x44, 0x44, 0xff}
	colorDead  = color.RGBA{0x22, 0x22, 0x22, 0xff}

	cellParams = map[Cell]CellParams{
		CellEmpty: {colorSea, colorEmpty, 0.},
		CellMiss:  {colorSea, colorMist, 0.25},
		CellMist:  {colorMist, colorEmpty, 0.},
		CellHide:  {colorMist, colorEmpty, 0.},
		CellShip:  {colorShip, colorEmpty, 0.},
		CellFire:  {color.RGBA{0x88, 0x22, 0x22, 0xff}, color.RGBA{0xff, 0x88, 0x22, 0x88}, 0.45},
		CellSunk:  {colorSea, colorDead, 0.55},
		CellOily:  {colorSea, colorEmpty, 0.},
	}

	fillImage = func() *ebiten.Image {
		img := ebiten.NewImageWithOptions(image.Rect(-1, -1, 1, 1), nil)
		img.Fill(color.RGBA{0xff, 0xff, 0xff, 0xff})
		return img
	}()
)

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

func (xy XY) String() string {
	return fmt.Sprintf("%c%c", 'A'+xy.X, '1'+xy.Y)
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
	Tick        int64
	LastUpdate  int64 // the tick when was the last update on the board.
	Boards      [2]*Board
	WhoseTurn   Side
	Message     string
	Error       error // terminating error.
	CursorSelf  XY
	CursorPeer  XY
	PeerToHit   XY // Where is the spot peer wants to hit.
	LastPeerHit XY // The successful one.

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
		Message:   "Note: ships can only touch by corners",
		cellImage: ebiten.NewImage(cellSize, cellSize),
	}
	g.Boards = [2]*Board{NewBoard(g, SideSelf), NewBoard(g, SidePeer)}
	return g
}

func (g *Game) init() error {
	for _, b := range g.Boards {
		if err := b.addRandomShips(30); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) Update() error {
	if g.Tick == 0 {
		if err := g.init(); err != nil {
			g.Error = err
		}
	}
	g.Tick++
	if g.Error != nil {
		return ebiten.Termination
	}
	if err := g.update(); err != nil {
		g.Error = err
	}
	return nil
}

func (g *Game) update() error {
	if err := g.handleKeys(); err != nil {
		return err
	}
	if err := g.handleTouches(); err != nil {
		return err
	}
	if err := g.handleMouse(); err != nil {
		return err
	}

	// Handle peer activity.
	if g.WhoseTurn == SideSelf || g.Tick-g.LastUpdate < peerTicksPerAct {
		return nil
	}
	g.LastUpdate = g.Tick
	if g.CursorPeer != g.PeerToHit {
		// moving peer cursor
		g.CursorPeer.X += sign(g.PeerToHit.X - g.CursorPeer.X)
		g.CursorPeer.Y += sign(g.PeerToHit.Y - g.CursorPeer.Y)
		return nil
	}
	if g.Boards[SideSelf].hitCell(g.PeerToHit) {
		g.LastPeerHit = g.PeerToHit
		if g.Boards[SideSelf].Lives == 0 {
			// The last ship is dead!
			return fmt.Errorf("The peer has won the game!")
		}
		if err := g.peerToHit(); err != nil {
			return err
		}
	} else {
		g.WhoseTurn = SideSelf
	}
	return nil
}

func sign(v int) int {
	if v < 0 {
		return -1
	} else if v > 0 {
		return 1
	}
	return 0
}

func (g *Game) handleKeys() error {
	g.keys = inpututil.AppendJustReleasedKeys(g.keys[:0])
	for _, k := range g.keys {
		if k == ebiten.KeyQ {
			// TODO: remove this.
			// Special handling even during peer turn.
			return fmt.Errorf("Stopped by player")
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
			if g.Boards[SidePeer].hitCell(g.CursorSelf) {
				if g.Boards[SidePeer].Lives == 0 {
					return fmt.Errorf("You have won the game!")
				}
			} else {
				if err := g.peerToHit(); err != nil {
					return err
				}
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
		g.CursorSelf.X = pos2Cell(tx, Ncells+1, Ncells)
		g.CursorSelf.Y = pos2Cell(ty, 1, Ncells)
	}
	for _, t := range g.killedTouches {
		// TODO: if touch is outside the board, do not hit it.
		tx, ty := inpututil.TouchPositionInPreviousTick(t)
		g.CursorSelf.X = pos2Cell(tx, Ncells+1, Ncells)
		g.CursorSelf.Y = pos2Cell(ty, 1, Ncells)
		if g.Boards[SidePeer].hitCell(g.CursorSelf) {
			if g.Boards[SidePeer].Lives == 0 {
				return fmt.Errorf("You have won the game!")
			}
		} else {
			return g.peerToHit()
		}
	}
	return nil
}

func (g *Game) handleMouse() error {
	if g.WhoseTurn != SideSelf {
		return nil
	}
	cx, cy := ebiten.CursorPosition()
	pressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	justReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	if pressed || justReleased {
		// Draw game cursor.
		g.CursorSelf.X = pos2Cell(cx, Ncells+1, Ncells)
		g.CursorSelf.Y = pos2Cell(cy, 1, Ncells)
	}
	if justReleased {
		if g.Boards[SidePeer].hitCell(g.CursorSelf) {
			if g.Boards[SidePeer].Lives == 0 {
				return fmt.Errorf("You have won the game!")
			}
		} else {
			return g.peerToHit()
		}
	}
	return nil
}

// peerToHit is to find where peer wants to hit.
func (g *Game) peerToHit() error {
	g.WhoseTurn = SidePeer
	g.LastUpdate = g.Tick
	if g.Boards[SideSelf].Cells[g.LastPeerHit.Y][g.LastPeerHit.X] == CellFire {
		xys := g.peerToHitShipMore()
		if len(xys) == 0 {
			return fmt.Errorf("cannot find next hit point after %s", g.LastPeerHit)
		}
		g.PeerToHit = xys[rand.Intn(len(xys))]
		return nil
	}
	return g.huntLargestStrategy()
	// return g.uniformStrategy()
}

// uniformStrategy uniformly choose a random cell to hit.
func (g *Game) uniformStrategy() error {
	// The previous attempt was a miss, or the ship was sunk.
	g.PeerToHit.X = rand.Intn(Ncells)
	g.PeerToHit.Y = rand.Intn(Ncells)
	for i := 0; i < Ncells*Ncells; i++ {
		c := g.Boards[SideSelf].Cells[g.PeerToHit.Y][g.PeerToHit.X]
		if c == CellEmpty || c == CellShip {
			return nil
		}
		// check the next cell
		g.PeerToHit.X++
		if g.PeerToHit.X >= Ncells {
			g.PeerToHit.X = 0
			g.PeerToHit.Y++
			if g.PeerToHit.Y >= Ncells {
				g.PeerToHit.Y = 0
			}
		}
	}
	// All cells are not suitable!
	return fmt.Errorf("all cells are hit already")
}

// huntLargestStrategy hunts for the largest ship.
func (g *Game) huntLargestStrategy() error {
	largest := maxShipSize
	b := g.Boards[SideSelf]
	for ; largest > 0; largest-- {
		if b.Ships[largest-1] > 0 {
			break
		}
	}
	if largest <= 0 {
		return fmt.Errorf("could not determine largest ship")
	}
	cellWeight := make(map[XY]int)
	markSlice := func(i0, i1 int, f func(i, w int)) {
		if i0 == -1 {
			return
		}
		if i1-i0 < largest {
			return
		}
		// 0123456 <-- indices
		// 123321  <-- weight
		// log.Printf("markSlice(%d, %d)", i0, i1)
		for i := i0; i < i1; i++ {
			w := largest
			if w1 := i - i0 + 1; w1 < w {
				w = w1
			}
			if w2 := i1 - i; w2 < w {
				w = w2
			}
			f(i, w)
		}
	}
	// Scan along X.
	for y := 0; y < Ncells; y++ {
		xStart := -1
		x := 0
		fx := func(i, w int) {
			xy := XY{i, y}
			w0 := cellWeight[xy]
			cellWeight[xy] = w0 + w
			// log.Printf("weight %s: %d + %d -> %d", xy, w0, w, cellWeight[xy])
		}
		for ; x < Ncells; x++ {
			switch c := b.Cells[y][x]; c {
			case CellEmpty, CellShip:
				if xStart == -1 {
					xStart = x
				}
			default:
				markSlice(xStart, x, fx)
				xStart = -1
			}
		}
		markSlice(xStart, x, fx)
	}
	// Scan along Y.
	for x := 0; x < Ncells; x++ {
		yStart := -1
		y := 0
		fy := func(i, w int) {
			xy := XY{x, i}
			w0 := cellWeight[xy]
			cellWeight[xy] = w0 + w
			// log.Printf("weight %s: %d + %d -> %d", xy, w0, w, cellWeight[xy])
		}
		for ; y < Ncells; y++ {
			switch c := b.Cells[y][x]; c {
			case CellEmpty, CellShip:
				if yStart == -1 {
					yStart = y
				}
			default:
				markSlice(yStart, y, fy)
				yStart = -1
			}
		}
		markSlice(yStart, y, fy)
	}
	// Find cells with max count in the map.
	maxW := 0
	var cells []XY
	for xy, w := range cellWeight {
		if w < maxW {
			continue
		}
		if w > maxW {
			cells = cells[:0]
			maxW = w
		}
		cells = append(cells, xy)
	}
	// log.Printf("total suitable cells: %d; maxW(%d) cells(%d): %v", len(cellWeight), maxW, len(cells), cells)
	if len(cells) == 0 {
		return fmt.Errorf("cannot find cells for the largest ship")
	}
	i := rand.Intn(len(cells))
	g.PeerToHit = cells[i]
	return nil
}

func (g *Game) peerToHitShipMore() []XY {
	b := g.Boards[SideSelf]
	only := false
	check := func(xys *[]XY, xy XY) bool {
		switch c := b.Cells[xy.Y][xy.X]; c {
		case CellFire:
			only = true
			return true
		case CellEmpty, CellShip:
			*xys = append(*xys, xy)
		}
		return false
	}
	xs := make([]XY, 0, 4)
	for dx := -1; dx < 2; dx += 2 {
		for x := g.LastPeerHit.X + dx; x >= 0 && x < Ncells && check(&xs, XY{x, g.LastPeerHit.Y}); x += dx {
		}
	}
	if only {
		return xs
	}
	ys := make([]XY, 0, 4)
	for dy := -1; dy < 2; dy += 2 {
		for y := g.LastPeerHit.Y + dy; y >= 0 && y < Ncells && check(&ys, XY{g.LastPeerHit.X, y}); y += dy {
		}
	}
	if only {
		return ys
	}
	return append(xs, ys...)
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
	msg := g.Message
	if g.Error != nil {
		msg = g.Error.Error()
	}
	if msg != "" {
		text.Draw(screen, msg, &text.GoTextFace{
			Source: ptSansFontSource,
			Size:   cellSize * 0.8,
		}, g.textInXY(Ncells, -1, SideSelf))
	}
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
