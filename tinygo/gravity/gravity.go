package main

import (
	"image/color"
	"machine"
	"math"
	"time"

	"tinygo-sample/gravity/bmi270"
	axp192 "tinygo.org/x/drivers/axp192/m5stack-core2-axp192"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

const (
	imgW      = 60
	imgH      = 60
	maxOffset = 80
	screenW   = 320
	screenH   = 240
	textW     = 240
	textH     = 30
)

var (
	white = color.RGBA{255, 255, 255, 255}
	black = color.RGBA{0, 0, 0, 255}
)

var textBuffer [textW * textH]uint16

func main() {
	time.Sleep(2 * time.Second)
	println("--- Microcontroller Booted ---")

	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40e6,
	})
	println("SPI configured")

	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400e3,
		SDA:       machine.SDA0_PIN,
		SCL:       machine.SCL0_PIN,
	})
	if err != nil {
		println("I2C configured error:", err.Error())
	} else {
		println("I2C configured successfully")
	}

	axp := axp192.New(i2c)
	err = axp.Configure(axp192.Config{})
	if err != nil {
		println("AXP Configure err:", err.Error())
	}
	led := axp.LED
	led.Low()
	println("AXP configured")

	display := ili9341.NewSPI(
		machine.SPI2,
		machine.LCD_DC_PIN,
		machine.LCD_SS_PIN,
		machine.NoPin,
	)

	display.Configure(ili9341.Config{
		Width:            320,
		Height:           240,
		DisplayInversion: true,
	})

	display.SetRotation(ili9341.Rotation0Mirror)
	println("Display configured")

	time.Sleep(100 * time.Millisecond)

	// BMI270の初期化
	println("Initializing BMI270...")
	imu := bmi270.New(i2c)
	err = imu.Init()
	if err != nil {
		println("BMI270 init err:", err)
	} else {
		println("BMI270 initialized successfully")
	}

	// バイブレーションを止める（LDO3を無効化: 0x45）
	println("Disabling vibration motor...")
	err = i2c.Tx(0x34, []byte{0x12, 0x45}, nil)
	println("LDO3 disable err:", err)
	time.Sleep(10 * time.Millisecond)

	renderTextToBuffer()

	var ax, ay float64
	count := 0

	for {
		x, y, _, err := imu.ReadAcceleration()
		if err == nil {
			ax = x
			ay = y
		}

		if count%60 == 0 {
			println("ax:", int(ax*100), "ay:", int(ay*100), "err:", err)
		}
		count++

		clampX := ax
		if clampX > 2.0 {
			clampX = 2.0
		} else if clampX < -2.0 {
			clampX = -2.0
		}
		clampX /= 2.0

		clampY := ay
		if clampY > 2.0 {
			clampY = 2.0
		} else if clampY < -2.0 {
			clampY = -2.0
		}
		clampY /= 2.0

		gx := screenW/2 + int(clampX*maxOffset)
		gy := screenH/2 + int(clampY*maxOffset)

		angleDeg := math.Atan2(ax, ay) * 180.0 / math.Pi

		len := math.Sqrt(ax*ax + ay*ay)
		offsetDist := 60.0
		ux, uy := 0.0, -1.0
		if len > 0.01 {
			ux = -ax / len
			uy = -ay / len
		}
		tx := gx + int(ux*offsetDist)
		ty := gy + int(uy*offsetDist)

		display.FillScreen(white)

		drawRotatedImage(display, gx, gy, angleDeg)
		drawRotatedText(display, tx, ty, angleDeg)

		time.Sleep(16 * time.Millisecond)
	}
}

func renderTextToBuffer() {
	for i := range textBuffer {
		textBuffer[i] = 0xFFFF
	}

	bufDisplay := &bufferDisplay{
		buffer: textBuffer[:],
		width:  textW,
		height: textH,
	}

	tinyfont.WriteLine(bufDisplay, &freemono.Regular9pt7b, 0, 20, "Hello, TinyGo World", black)
}

type bufferDisplay struct {
	buffer []uint16
	width  int
	height int
}

func (d *bufferDisplay) SetPixel(x, y int16, c color.RGBA) {
	if x >= 0 && int(x) < d.width && y >= 0 && int(y) < d.height {
		r := uint16(c.R) >> 3
		g := uint16(c.G) >> 2
		b := uint16(c.B) >> 3
		d.buffer[int(y)*d.width+int(x)] = (r << 11) | (g << 5) | b
	}
}

func (d *bufferDisplay) Display() error {
	return nil
}

func (d *bufferDisplay) Size() (int16, int16) {
	return int16(d.width), int16(d.height)
}

func drawRotatedImage(display *ili9341.Device, cx, cy int, angleDeg float64) {
	angleRad := angleDeg * math.Pi / 180.0
	cosA := math.Cos(angleRad)
	sinA := math.Sin(angleRad)

	halfW := imgW / 2
	halfH := imgH / 2

	for dy := -halfH; dy < halfH; dy++ {
		for dx := -halfW; dx < halfW; dx++ {
			srcX := int(float64(dx)*cosA+float64(dy)*sinA) + halfW
			srcY := int(-float64(dx)*sinA+float64(dy)*cosA) + halfH

			if srcX >= 0 && srcX < imgW && srcY >= 0 && srcY < imgH {
				idx := srcY*imgW + srcX
				rgb565 := gopherData[idx]

				r := uint8(((rgb565 >> 11) & 0x1F) << 3)
				g := uint8(((rgb565 >> 5) & 0x3F) << 2)
				b := uint8((rgb565 & 0x1F) << 3)

				c := color.RGBA{r, g, b, 255}
				display.SetPixel(int16(cx+dx), int16(cy+dy), c)
			}
		}
	}
}

func drawRotatedText(display *ili9341.Device, cx, cy int, angleDeg float64) {
	angleRad := angleDeg * math.Pi / 180.0
	cosA := math.Cos(angleRad)
	sinA := math.Sin(angleRad)

	halfW := textW / 2
	halfH := textH / 2

	for dy := -halfH; dy < halfH; dy++ {
		for dx := -halfW; dx < halfW; dx++ {
			srcX := int(float64(dx)*cosA+float64(dy)*sinA) + halfW
			srcY := int(-float64(dx)*sinA+float64(dy)*cosA) + halfH

			if srcX >= 0 && srcX < textW && srcY >= 0 && srcY < textH {
				idx := srcY*textW + srcX
				rgb565 := textBuffer[idx]

				if rgb565 != 0xFFFF {
					r := uint8(((rgb565 >> 11) & 0x1F) << 3)
					g := uint8(((rgb565 >> 5) & 0x3F) << 2)
					b := uint8((rgb565 & 0x1F) << 3)

					c := color.RGBA{r, g, b, 255}
					display.SetPixel(int16(cx+dx), int16(cy+dy), c)
				}
			}
		}
	}
}
