package vm

import "github.com/joushou/gocnc/gcode"
import "fmt"

//
// Export machine code
//

func (vm *Machine) Export() *gcode.Document {
	var (
		lastFeedrate, lastSpindleSpeed, lastX, lastY, lastZ         float64
		spindleEnabled, spindleClockwise, mistCoolant, floodCoolant bool
		lastMoveMode, lastMovePlane                                 float64
		doc                                                         gcode.Document
	)

	shortBlock := func(n gcode.Node) {
		var block gcode.Block
		block.AppendNode(n)
		doc.AppendBlock(block)
	}

	shortBlock(&gcode.Comment{"Exported by gocnc/vm", false})

	var headerBlock gcode.Block
	headerBlock.AppendNode(&gcode.Word{'G', 21})
	//	headerBlock.AppendNode(&gcode.Word{'G', 40})
	headerBlock.AppendNode(&gcode.Word{'G', 90})
	headerBlock.AppendNode(&gcode.Word{'G', 94})
	doc.AppendBlock(headerBlock)

	for _, pos := range vm.posStack {
		s := pos.state
		var moveMode, movePlane float64

		// fetch move mode
		switch s.moveMode {
		case moveModeInitial:
			continue
		case moveModeRapid:
			moveMode = 0
		case moveModeLinear:
			moveMode = 1
		case moveModeCWArc:
			moveMode = 2
		case moveModeCCWArc:
			moveMode = 3
		}

		// fetch move plane
		switch s.movePlane {
		case planeXY:
			movePlane = 17
		case planeXZ:
			movePlane = 18
		case planeYZ:
			movePlane = 19
		}

		// handle spindle
		if s.spindleEnabled != spindleEnabled || s.spindleClockwise != spindleClockwise {
			if s.spindleEnabled && s.spindleClockwise {
				shortBlock(&gcode.Word{'M', 3})
			} else if s.spindleEnabled && !s.spindleClockwise {
				shortBlock(&gcode.Word{'M', 4})
			} else if !s.spindleEnabled {
				shortBlock(&gcode.Word{'M', 5})
			}
			spindleEnabled, spindleClockwise = s.spindleEnabled, s.spindleClockwise
			lastMoveMode = -1 // M-codes clear stuff...
		}

		// handle coolant
		if s.floodCoolant != floodCoolant || s.mistCoolant != mistCoolant {

			if (floodCoolant == true && s.floodCoolant == false) ||
				(mistCoolant == true && s.mistCoolant == false) {
				// We can only disable both coolants simultaneously, so kill it and reenable one if needed
				shortBlock(&gcode.Word{'M', 9})
			}
			if s.floodCoolant {
				shortBlock(&gcode.Word{'M', 8})
			} else if s.mistCoolant {
				shortBlock(&gcode.Word{'M', 7})
			}
			lastMoveMode = -1 // M-codes clear stuff...
		}

		// handle feedrate and spindle speed
		if s.moveMode != moveModeRapid {
			if s.feedrate != lastFeedrate {
				shortBlock(&gcode.Word{'F', s.feedrate})
				lastFeedrate = s.feedrate
			}
			if s.spindleSpeed != lastSpindleSpeed {
				shortBlock(&gcode.Word{'S', s.spindleSpeed})
				lastSpindleSpeed = s.spindleSpeed
			}
		}

		var moveBlock gcode.Block

		// handle move plane
		if movePlane != lastMovePlane {
			moveBlock.AppendNode(&gcode.Word{'G', movePlane})
			lastMovePlane = movePlane
		}

		// handle move mode
		if s.moveMode == moveModeCWArc || s.moveMode == moveModeCCWArc || moveMode != lastMoveMode {
			moveBlock.AppendNode(&gcode.Word{'G', moveMode})
			lastMoveMode = moveMode
		}

		// handle move
		if pos.x != lastX {
			moveBlock.AppendNode(&gcode.Word{'X', pos.x})
			lastX = pos.x
		}
		if pos.y != lastY {
			moveBlock.AppendNode(&gcode.Word{'Y', pos.y})
			lastY = pos.y
		}
		if pos.z != lastZ {
			moveBlock.AppendNode(&gcode.Word{'Z', pos.z})
			lastZ = pos.z
		}

		// handle arc
		if s.moveMode == moveModeCWArc || s.moveMode == moveModeCCWArc {
			if pos.i != 0 {
				moveBlock.AppendNode(&gcode.Word{'I', pos.i})
			}
			if pos.j != 0 {
				moveBlock.AppendNode(&gcode.Word{'J', pos.j})
			}
			if pos.k != 0 {
				moveBlock.AppendNode(&gcode.Word{'K', pos.k})
			}
			if pos.rot != 1 {
				moveBlock.AppendNode(&gcode.Word{'P', float64(pos.rot)})
			}
		}

		// put on slice
		if len(moveBlock.Nodes) > 0 {
			doc.AppendBlock(moveBlock)
		}
	}
	return &doc
}

//
// Dump moves in (sort of) human readable format
//
func (vm *Machine) Dump() {
	for _, m := range vm.posStack {
		switch m.state.moveMode {
		case moveModeInitial:
			fmt.Printf("initial pos, ")
		case moveModeRapid:
			fmt.Printf("rapid move, ")
		case moveModeLinear:
			fmt.Printf("linear move, ")
		case moveModeCWArc:
			fmt.Printf("clockwise arc, ")
		case moveModeCCWArc:
			fmt.Printf("counterclockwise arc, ")
		}

		fmt.Printf("feedrate: %f, ", m.state.feedrate)
		fmt.Printf("spindle: %f, ", m.state.spindleSpeed)
		fmt.Printf("X: %f, Y: %f, Z: %f, I: %f, J: %f, K: %f\n", m.x, m.y, m.z, m.i, m.j, m.k)
	}
}
