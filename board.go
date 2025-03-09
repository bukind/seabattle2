package main

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Board struct {
	Game  *Game
	Side  Side
	Lives int
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
					b.Lives += s
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
		case CellHide, CellShip, CellFire, CellSunk:
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
		case CellHide, CellShip, CellFire, CellSunk:
			return false
		}
	}
	return true
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
		if sunk := b.isShipSunk(xy.X, xy.Y); len(sunk) > 0 {
			for _, xy := range sunk {
				b.Cells[xy.Y][xy.X] = CellSunk
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
