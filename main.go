package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

func main() {
	programName := os.Args[1]
	chip := NewChip8()

	fmt.Println("Loading application...")
	chip.loadApplication(programName)
	fmt.Println("Loaded application.")

	gfx(&chip)
}

func gfx(c8 *Chip8) {
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	s.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorWhite))
	s.Clear()

	quit := make(chan struct{})
	go func() {
		for {
			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape:
					close(quit)
					return
				case tcell.KeyCtrlL:
					s.Sync()
				}
				switch ev.Rune() {
				case '1':
					c8.Key = [16]uint8{}
					c8.Key[0x1] = 1
				case '2':
					c8.Key = [16]uint8{}
					c8.Key[0x2] = 1
				case '3':
					c8.Key = [16]uint8{}
					c8.Key[0x3] = 1
				case '4':
					c8.Key = [16]uint8{}
					c8.Key[0xC] = 1
				case 'q':
					c8.Key = [16]uint8{}
					c8.Key[0x4] = 1
				case 'w':
					c8.Key = [16]uint8{}
					c8.Key[0x5] = 1
				case 'e':
					c8.Key = [16]uint8{}
					c8.Key[0x6] = 1
				case 'r':
					c8.Key = [16]uint8{}
					c8.Key[0xD] = 1
				case 'a':
					c8.Key = [16]uint8{}
					c8.Key[0x7] = 1
				case 's':
					c8.Key = [16]uint8{}
					c8.Key[0x8] = 1
				case 'd':
					c8.Key = [16]uint8{}
					c8.Key[0x9] = 1
				case 'f':
					c8.Key = [16]uint8{}
					c8.Key[0xE] = 1
				case 'z':
					c8.Key = [16]uint8{}
					c8.Key[0xA] = 1
				case 'x':
					c8.Key = [16]uint8{}
					c8.Key[0x0] = 1
				case 'c':
					c8.Key = [16]uint8{}
					c8.Key[0xB] = 1
				case 'v':
					c8.Key = [16]uint8{}
					c8.Key[0xF] = 1
				}
			case *tcell.EventResize:
				s.Sync()
			}
		}
	}()

loop:
	for {
		select {
		case <-quit:
			break loop
		case <-time.After(time.Millisecond * 16):
		}

		for row := 0; row < screenHeight; row++ {
			for col := 0; col < screenWidth; col++ {
				isOn := c8.GFX[(row*screenWidth)+col] == 1
				var cellToUse tcell.Style
				if isOn {
					onPixel := tcell.NewHexColor(0xFFFFFF)
					cellToUse = tcell.StyleDefault.Background(onPixel)
				} else {
					offPixel := tcell.NewHexColor(0x000000)
					cellToUse = tcell.StyleDefault.Background(offPixel)
				}
				s.SetCell(col, row, cellToUse)
			}
		}
		s.Show()
		c8.emulateCycle()
	}

	s.Fini()
}

type Chip8 struct {
	Opcode     uint16
	Memory     [4096]uint8
	V          [16]uint8
	I          uint16
	PC         uint16
	GFX        [screenWidth * screenHeight]uint8
	DelayTimer uint8
	SoundTimer uint8
	Stack      [16]uint16
	SP         uint16
	Key        [16]uint8
}

const fontSetSize = 80
const screenWidth = 64
const screenHeight = 32

var fontSet [fontSetSize]uint8 = [fontSetSize]uint8{
	0xF0, 0x90, 0x90, 0x90, 0xF0, // 0
	0x20, 0x60, 0x20, 0x20, 0x70, // 1
	0xF0, 0x10, 0xF0, 0x80, 0xF0, // 2
	0xF0, 0x10, 0xF0, 0x10, 0xF0, // 3
	0x90, 0x90, 0xF0, 0x10, 0x10, // 4
	0xF0, 0x80, 0xF0, 0x10, 0xF0, // 5
	0xF0, 0x80, 0xF0, 0x90, 0xF0, // 6
	0xF0, 0x10, 0x20, 0x40, 0x40, // 7
	0xF0, 0x90, 0xF0, 0x90, 0xF0, // 8
	0xF0, 0x90, 0xF0, 0x10, 0xF0, // 9
	0xF0, 0x90, 0xF0, 0x90, 0x90, // A
	0xE0, 0x90, 0xE0, 0x90, 0xE0, // B
	0xF0, 0x80, 0x80, 0x80, 0xF0, // C
	0xE0, 0x90, 0x90, 0x90, 0xE0, // D
	0xF0, 0x80, 0xF0, 0x80, 0xF0, // E
	0xF0, 0x80, 0xF0, 0x80, 0x80, // F
}

func NewChip8() Chip8 {
	var mem [4096]uint8
	for i := 0; i < fontSetSize; i++ {
		mem[i] = fontSet[i]
	}
	return Chip8{
		PC:     0x200,
		Memory: mem,
	}
}

func (c *Chip8) loadApplication(filename string) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("unable to read file %v", filename)
	}
	for i, b := range bytes {
		c.Memory[512+i] = b
	}
}

func (c *Chip8) emulateCycle() {
	c.fetchOpcode()
	skip := c.executeOpcode()
	if skip {
		return
	}
	c.updateTimers()
}

func (c *Chip8) fetchOpcode() {
	c.Opcode = uint16(c.Memory[c.PC])<<8 | uint16(c.Memory[c.PC+1])
}

// Returns true if cycle should be skipped
func (c *Chip8) executeOpcode() bool {
	switch c.Opcode & 0xF000 {
	case 0x0000:
		switch c.Opcode & 0x000F {
		case 0x0000:
			// 0x00E0: Clears the screen
			c.GFX = [2048]uint8{}
			c.PC += 2
		case 0x000E:
			// 0x00EE: Returns from subroutine
			c.SP--
			c.PC = c.Stack[c.SP]
			c.PC += 2
		default:
			panicUnknownOpcode(c.Opcode)
		}
	case 0x1000:
		// 0x1NNN: Jumps to address NNN
		c.PC = c.Opcode & 0x0FFF
	case 0x2000:
		// 0x2NNN: Calls subroutine at NNN.
		c.Stack[c.SP] = c.PC
		c.SP++
		c.PC = c.Opcode & 0x0FFF
	case 0x3000:
		// 0x3XNN: Skips the next instruction if VX equals NN
		if c.V[(c.Opcode&0x0F00)>>8] == (uint8(c.Opcode) & 0x00FF) {
			c.PC += 4
		} else {
			c.PC += 2
		}
	case 0x4000:
		// 0x4XNN: Skips the next instruction if VX doesn't equal NN
		if c.V[(c.Opcode&0x0F00)>>8] != (uint8(c.Opcode) & 0x00FF) {
			c.PC += 4
		} else {
			c.PC += 2
		}
	case 0x5000:
		// 0x5XY0: Skips the next instruction if VX equals VY.
		if c.V[(c.Opcode&0x0F00)>>8] != c.V[(uint8(c.Opcode)&0x00F0)>>4] {
			c.PC += 4
		} else {
			c.PC += 2
		}
	case 0x6000:
		// 0x6XNN: Sets VX to NN.
		c.V[(c.Opcode&0x0F00)>>8] = uint8(c.Opcode) & 0x00FF
		c.PC += 2
	case 0x7000:
		// 0x7XNN: Adds NN to VX.
		c.V[(c.Opcode&0x0F00)>>8] += uint8(c.Opcode) & 0x00FF
		c.PC += 2
	case 0x8000:
		switch c.Opcode & 0x000F {
		case 0x0000:
			// 0x8XY0: Sets VX to the value of VY
			c.V[(c.Opcode&0x0F00)>>8] = c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2
		case 0x0001:
			// 0x8XY1: Sets VX to "VX OR VY"
			c.V[(c.Opcode&0x0F00)>>8] |= c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2
		case 0x0002:
			// 0x8XY2: Sets VX to "VX AND VY"
			c.V[(c.Opcode&0x0F00)>>8] &= c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2
		case 0x0003:
			// 0x8XY3: Sets VX to "VX XOR VY"
			c.V[(c.Opcode&0x0F00)>>8] ^= c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2
		case 0x0004:
			// 0x8XY4: Adds VY to VX. VF is set to 1 when there's a carry, and to 0 when there isn't
			if c.V[(c.Opcode&0x00F0)>>4] > (0xFF - c.V[(c.Opcode&0x0F00)>>8]) {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
			c.V[(c.Opcode&0x0F00)>>8] += c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2
		case 0x0005:
			// 0x8XY5: VY is subtracted from VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if c.V[(c.Opcode&0x00F0)>>4] > c.V[(c.Opcode&0x0F00)>>8] {

				c.V[0xF] = 0
			} else {

				c.V[0xF] = 1
			}
			c.V[(c.Opcode&0x0F00)>>8] -= c.V[(c.Opcode&0x00F0)>>4]
			c.PC += 2

		case 0x0006:
			// 0x8XY6: Shifts VX right by one. VF is set to the value of the least significant bit of VX before the shift
			c.V[0xF] = c.V[(c.Opcode&0x0F00)>>8] & 0x1
			c.V[(c.Opcode&0x0F00)>>8] >>= 1
			c.PC += 2
		case 0x0007:
			// 0x8XY7: Sets VX to VY minus VX. VF is set to 0 when there's a borrow, and 1 when there isn't
			if c.V[(c.Opcode&0x0F00)>>8] > c.V[(c.Opcode&0x00F0)>>4] {
				c.V[0xF] = 0
			} else {
				c.V[0xF] = 1
			}
			c.V[(c.Opcode&0x0F00)>>8] = c.V[(c.Opcode&0x00F0)>>4] - c.V[(c.Opcode&0x0F00)>>8]
			c.PC += 2
		case 0x000E:
			// 0x8XYE: Shifts VX left by one. VF is set to the value of the most significant bit of VX before the shift
			c.V[0xF] = c.V[(c.Opcode&0x0F00)>>8] >> 7
			c.V[(c.Opcode&0x0F00)>>8] <<= 1
			c.PC += 2
		default:
			panicUnknownOpcode(c.Opcode)
		}
	case 0x9000:
		// 0x9XY0: Skips the next instruction if VX doesn't equal VY
		if c.V[(c.Opcode&0x0F00)>>8] != c.V[(c.Opcode&0x00F0)>>4] {
			c.PC += 4
		} else {
			c.PC += 2
		}
	case 0xA000:
		// ANNN: Sets I to the address NNN
		c.I = c.Opcode & 0x0FFF
		c.PC += 2
	case 0xB000:
		// BNNN: Jumps to the address NNN plus V0
		c.PC = (c.Opcode & 0x0FFF) + uint16(c.V[0])
	case 0xC000:
		// CXNN: Sets VX to a random number and NN
		c.V[(c.Opcode&0x0F00)>>8] = randomByte() & uint8(c.Opcode&0x00FF)
		c.PC += 2
	case 0xD000:
		// DXYN: Draws a sprite at coordinate (VX, VY) that has a width of 8 pixels and a height of N pixels.
		x := uint16(c.V[(c.Opcode&0x0F00)>>8])
		y := uint16(c.V[(c.Opcode&0x00F0)>>4])
		height := uint16(c.Opcode & 0x000F)
		var pixel uint16

		c.V[0xF] = 0
		for yline := uint16(0); yline < height; yline++ {
			pixel = uint16(c.Memory[c.I+yline])
			for xline := uint16(0); xline < 8; xline++ {
				if (pixel & (0x80 >> xline)) != 0 {
					if c.GFX[x+xline+((y+yline)*screenWidth)] == 1 {
						c.V[0xF] = 1
					}
					c.GFX[x+xline+((y+yline)*screenWidth)] ^= 1
				}
			}
		}

		c.PC += 2

	case 0xE000:
		switch c.Opcode & 0x00FF {
		case 0x009E:
			// EX9E: Skips the next instruction if the key stored in VX is pressed
			if c.Key[c.V[(c.Opcode&0x0F00)>>8]] != 0 {
				c.PC += 4
			} else {
				c.PC += 2
			}
		case 0x00A1:
			// EXA1: Skips the next instruction if the key stored in VX isn't pressed
			if c.Key[c.V[(c.Opcode&0x0F00)>>8]] == 0 {
				c.PC += 4
			} else {
				c.PC += 2
			}
		default:
			panicUnknownOpcode(c.Opcode)
		}
	case 0xF000:
		switch c.Opcode & 0x00FF {
		case 0x0007:
			// FX07: Sets VX to the value of the delay timer
			c.V[(c.Opcode&0x0F00)>>8] = c.DelayTimer
			c.PC += 2
		case 0x000A:
			keyPress := false
			for i := uint8(0); i < 16; i++ {
				if c.Key[i] != 0 {
					c.V[(c.Opcode&0x0F00)>>8] = i
					keyPress = true
				}
			}
			if !keyPress {
				return true
			}
			c.PC += 2
		case 0x0015:
			// FX15: Sets the delay timer to VX
			c.DelayTimer = c.V[(c.Opcode&0x0F00)>>8]
			c.PC += 2
		case 0x0018:
			// FX18: Sets the sound timer to VX
			c.SoundTimer = c.V[(c.Opcode&0x0F00)>>8]
			c.PC += 2
		case 0x001E:
			// FX1E: Adds VX to I
			if c.I+uint16(c.V[(c.Opcode&0x0F00)>>8]) > 0xFFF {
				c.V[0xF] = 1
			} else {
				c.V[0xF] = 0
			}
			c.I += uint16(c.V[(c.Opcode&0x0F00)>>8])
			c.PC += 2
		case 0x0029:
			// FX29: Sets I to the location of the sprite for the character in VX. Characters 0-F (in hexadecimal) are represented by a 4x5 font
			c.I = uint16(c.V[(c.Opcode&0x0F00)>>8]) * 0x5
			c.PC += 2
		case 0x0033:
			// FX33: Stores the Binary-coded decimal representation of VX at the addresses I, I plus 1, and I plus 2
			c.Memory[c.I] = c.V[(c.Opcode&0x0F00)>>8] / 100
			c.Memory[c.I+1] = (c.V[(c.Opcode&0x0F00)>>8] / 10) % 10
			c.Memory[c.I+2] = (c.V[(c.Opcode&0x0F00)>>8] % 100) % 10
			c.PC += 2
		case 0x0055:
			// FX55: Stores V0 to VX in memory starting at address I
			for i := uint16(0); i <= ((c.Opcode & 0x0F00) >> 8); i++ {
				c.Memory[c.I+i] = c.V[i]
			}
			c.I += ((c.Opcode & 0x0F00) >> 8) + 1
			c.PC += 2
		case 0x0065:
			// FX65: Fills V0 to VX with values from memory starting at address I
			for i := uint16(0); i <= ((c.Opcode & 0x0F00) >> 8); i++ {
				c.V[i] = c.Memory[c.I+i]
			}
			c.I += ((c.Opcode & 0x0F00) >> 8) + 1
			c.PC += 2

		default:
			panicUnknownOpcode(c.Opcode)
		}
	default:
		panicUnknownOpcode(c.Opcode)
	}
	return false

}

func panicUnknownOpcode(opcode uint16) {
	log.Panicf("Unknown opcode %v", opcode)
}

func (c *Chip8) updateTimers() {
	if c.DelayTimer > 0 {
		c.DelayTimer--
	}
	if c.SoundTimer > 0 {
		c.SoundTimer--
	}
}

func randomByte() uint8 {
	rand.Seed(time.Now().UTC().UnixNano())
	randint := rand.Intn(math.MaxUint8)
	return uint8(randint)
}
