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
	stripH    = 20
)

var black = color.RGBA{0, 0, 0, 255}

// テキストスプライト（RGB565）
var textBuffer [textW * textH]uint16

// フレームバッファ（RGB565BE, 2bytes/pixel）
var fbRaw [screenW * screenH * 2]byte

func main() {
	time.Sleep(2 * time.Second)
	println("--- Microcontroller Booted ---")

	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40e6,
	})

	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 400e3,
		SDA:       machine.SDA0_PIN,
		SCL:       machine.SCL0_PIN,
	})

	axp := axp192.New(i2c)
	axp.Configure(axp192.Config{})
	axp.LED.Low()

	display := ili9341.NewSPI(
		machine.SPI2,
		machine.LCD_DC_PIN,
		machine.LCD_SS_PIN,
		machine.NoPin,
	)
	display.Configure(ili9341.Config{
		Width:            screenW,
		Height:           screenH,
		DisplayInversion: true,
	})
	display.SetRotation(ili9341.Rotation0Mirror)
	println("Display configured")

	// BMI270
	imu := bmi270.NewI2C(i2c, bmi270.Address)
	if err := imu.Configure(bmi270.DefaultConfig()); err != nil {
		println("BMI270 err:", err)
	}
	println("BMI270 ready")

	// テキストスプライト生成
	renderTextToBuffer()

	var ax, ay float64
	count := 0

	for {
		ix, iy, _, err := imu.ReadAcceleration()
		if err == nil {
			ax = float64(ix) / 1_000_000
			ay = float64(iy) / 1_000_000
		}

		if count%60 == 0 {
			println("ax:", int(ax*100), "ay:", int(ay*100), "err:", err)
		}
		count++

		// 位置計算
		clampX := clamp(ax, -2.0, 2.0) / 2.0
		clampY := clamp(ay, -2.0, 2.0) / 2.0
		gx := screenW/2 + int(clampX*maxOffset)
		gy := screenH/2 + int(clampY*maxOffset)

		// 角度計算
		angleRad := math.Atan2(ax, ay)
		cosA := math.Cos(angleRad)
		sinA := math.Sin(angleRad)

		// テキスト位置（Gopherの重力方向反対側）
		length := math.Sqrt(ax*ax + ay*ay)
		ux, uy := 0.0, -1.0
		if length > 0.01 {
			ux = -ax / length
			uy = -ay / length
		}
		tx := gx + int(ux*60)
		ty := gy + int(uy*60)

		// フレームバッファを白で塗りつぶし
		for i := range fbRaw {
			fbRaw[i] = 0xFF
		}

		// Gopher描画
		drawRotatedGopher(gx, gy, cosA, sinA)

		// テキスト描画
		drawRotatedText(tx, ty, cosA, sinA)

		// ストリップ分割転送
		rowBytes := screenW * 2
		for sy := 0; sy < screenH; sy += stripH {
			start := sy * rowBytes
			display.DrawRGBBitmap8(0, int16(sy), fbRaw[start:start+stripH*rowBytes], screenW, stripH)
		}

		time.Sleep(16 * time.Millisecond)
	}
}

func setFBPixel(x, y int, rgb565 uint16) {
	if x >= 0 && x < screenW && y >= 0 && y < screenH {
		offset := (y*screenW + x) * 2
		fbRaw[offset] = byte(rgb565 >> 8)
		fbRaw[offset+1] = byte(rgb565)
	}
}

func drawRotatedGopher(cx, cy int, cosA, sinA float64) {
	halfW := imgW / 2
	halfH := imgH / 2

	// 回転後のバウンディングボックス
	bboxHW := int(math.Abs(float64(halfW)*cosA)+math.Abs(float64(halfH)*sinA)) + 1
	bboxHH := int(math.Abs(float64(halfW)*sinA)+math.Abs(float64(halfH)*cosA)) + 1

	for dy := -bboxHH; dy < bboxHH; dy++ {
		for dx := -bboxHW; dx < bboxHW; dx++ {
			srcX := int(float64(dx)*cosA+float64(dy)*sinA) + halfW
			srcY := int(-float64(dx)*sinA+float64(dy)*cosA) + halfH

			if srcX >= 0 && srcX < imgW && srcY >= 0 && srcY < imgH {
				rgb565 := gopherData[srcY*imgW+srcX]
				if rgb565 != 0xFFFF {
					setFBPixel(cx+dx, cy+dy, rgb565)
				}
			}
		}
	}
}

func drawRotatedText(cx, cy int, cosA, sinA float64) {
	halfW := textW / 2
	halfH := textH / 2

	// 回転後のバウンディングボックス
	bboxHW := int(math.Abs(float64(halfW)*cosA)+math.Abs(float64(halfH)*sinA)) + 1
	bboxHH := int(math.Abs(float64(halfW)*sinA)+math.Abs(float64(halfH)*cosA)) + 1

	for dy := -bboxHH; dy < bboxHH; dy++ {
		for dx := -bboxHW; dx < bboxHW; dx++ {
			srcX := int(float64(dx)*cosA+float64(dy)*sinA) + halfW
			srcY := int(-float64(dx)*sinA+float64(dy)*cosA) + halfH

			if srcX >= 0 && srcX < textW && srcY >= 0 && srcY < textH {
				rgb565 := textBuffer[srcY*textW+srcX]
				if rgb565 != 0xFFFF {
					setFBPixel(cx+dx, cy+dy, rgb565)
				}
			}
		}
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

func (d *bufferDisplay) Display() error { return nil }

func (d *bufferDisplay) Size() (int16, int16) {
	return int16(d.width), int16(d.height)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
