package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Board struct {
	Game  *Game
	Side  Side
	Lives int
	Ships []int // number of ships of size = idx+1
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
		Ships: make([]int, maxShipSize),
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

func (b *Board) addRandomShips(retries int) error {
	num := 1
	for s := maxShipSize; s > 0; s-- {
		for n := 0; n < num; n++ {
			placed := false
			for attempt := 0; attempt < retries; attempt++ {
				if placed = b.placeShip(s); placed {
					b.Lives += s
					break
				}
			}
			if !placed {
				return fmt.Errorf("cannot place ship of size %d", s)
			}
			b.Ships[s-1]++
		}
		num++
	}
	// Replace working cells back to empty.
	cell := CellEmpty
	if b.Side == SidePeer {
		cell = CellMist
	}
	for y := 0; y < Ncells; y++ {
		for x := 0; x < Ncells; x++ {
			if b.Cells[y][x] == CellOily {
				b.Cells[y][x] = cell
			}
		}
	}
	return nil
}

func shipSeq(p0, p1 XY) func(yield func(XY) bool) {
	return func(yield func(XY) bool) {
		dx := sign(p1.X - p0.X)
		dy := sign(p1.Y - p0.Y)
		if dx == 0 {
			for yield(p0) {
				if p0.Y == p1.Y {
					return
				}
				p0.Y += dy
			}
		} else {
			for yield(p0) {
				if p0.X == p1.X {
					return
				}
				p0.X += dx
			}
		}
	}
}

// Temporary function to wrap range-over-func.
// It lacks early exit, i.e. it always generate all items in sequence.
// TODO: just drop it when we switch to go1.23.
func seqXY(rof func(func(XY) bool)) []XY {
	res := make([]XY, 0, 20)
	yield := func(xy XY) bool {
		res = append(res, xy)
		return true
	}
	rof(yield)
	return res
}

func (b *Board) placeShip(size int) bool {
	xSize := 0
	ySize := size - 1
	if r := rand.Intn(2); r != 0 {
		xSize = size - 1
		ySize = 0
	}
	p0 := XY{rand.Intn(Ncells - xSize), rand.Intn(Ncells - ySize)}
	p1 := XY{p0.X + xSize, p0.Y + ySize}
	for _, xy := range seqXY(shipSeq(p0, p1)) {
		switch c := b.Cells[xy.Y][xy.X]; c {
		case CellEmpty, CellMist:
			// ok
		default:
			return false
		}
	}
	cell := CellShip
	if b.Side == SidePeer {
		cell = CellHide
	}
	for _, xy := range seqXY(shipSeq(p0, p1)) {
		b.Cells[xy.Y][xy.X] = cell
	}
	for _, xy := range seqXY(b.markAround(p0, p1)) {
		switch c := b.Cells[xy.Y][xy.X]; c {
		case CellEmpty, CellMist:
			b.Cells[xy.Y][xy.X] = CellOily
		}
	}
	return true
}

func (b *Board) markAround(p0, p1 XY) func(func(XY) bool) {
	return func(yield func(xy XY) bool) {
		seqs := make([]func(func(XY) bool), 0, 4)
		log.Printf("markAround for %s %s", p0, p1)
		for _, y := range []int{p0.Y - 1, p1.Y + 1} {
			if y >= 0 && y < Ncells {
				log.Printf(" shipSeq %s %s", XY{p0.X, y}, XY{p1.X, y})
				seqs = append(seqs, shipSeq(XY{p0.X, y}, XY{p1.X, y}))
			}
		}
		for _, x := range []int{p0.X - 1, p1.X + 1} {
			if x >= 0 && x < Ncells {
				log.Printf(" shipSeq %s %s", XY{x, p0.Y}, XY{x, p1.Y})
				seqs = append(seqs, shipSeq(XY{x, p0.Y}, XY{x, p1.Y}))
			}
		}
		for _, seq := range seqs {
			for _, xy := range seqXY(seq) {
				if !yield(xy) {
					return
				}
			}
		}
	}
}

// return true if you can hit again.
func (b *Board) hitCell(xy XY) bool {
	switch c := b.Cells[xy.Y][xy.X]; c {
	case CellEmpty, CellMist:
		b.Cells[xy.Y][xy.X] = CellMiss
		return false
	case CellHide, CellShip:
		b.Cells[xy.Y][xy.X] = CellFire
		b.Lives--
		sunk := b.isShipSunk(xy.X, xy.Y)
		if len(sunk) > 0 {
			log.Printf("sunk %v", sunk)
			b.Ships[len(sunk)-1]--
			for _, xy := range sunk {
				b.Cells[xy.Y][xy.X] = CellSunk
			}
			if b.Side == SideSelf {
				for _, xy := range seqXY(b.markAround(sunk[0], sunk[len(sunk)-1])) {
					if c := b.Cells[xy.Y][xy.X]; c == CellEmpty {
						b.Cells[xy.Y][xy.X] = CellOily
					}
				}
			}
			g := b.Game
			g.Message = fmt.Sprintf("Sunk, remaining: %v, %v", g.Boards[0].Ships, g.Boards[1].Ships)
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
	result = append(result, XY{x0, y0})
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
	slices.SortFunc(result, func(a, b XY) int {
		if s := sign(a.Y - b.Y); s != 0 {
			return s
		}
		if s := sign(a.X - b.X); s != 0 {
			return s
		}
		return 0
	})
	return result
}
