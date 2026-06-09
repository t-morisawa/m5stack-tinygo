package main

import (
	"image/color"
	"machine"
	"time"

	axp192 "tinygo.org/x/drivers/axp192/m5stack-core2-axp192"
	"tinygo.org/x/drivers/ili9341"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

var (
	black = color.RGBA{0, 0, 0, 255}
	blue  = color.RGBA{0, 0, 255, 255}
	green = color.RGBA{0, 255, 0, 255}
)

func main() {
	// 1. 【超重要】シリアルモニタの接続を待つための猶予時間
	// マイコンが起動してからPCがシリアルを掴むまでラグがあるため、2秒待ちます
	time.Sleep(2 * time.Second)
	println("--- Microcontroller Booted ---") // これが表示されるか確認！

	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40e6,
	})
	println("SPI configured")

	// 2. 【修正】手動のI2Cバスリセット（GPIO操作）を一旦削除します
	// TinyGoの内部定義と競合してフリーズ（パニック）する原因になるためです。

	// 3. 【修正】I2Cの初期化を最もシンプルな形にする
	// TinyGoの `m5stack-core2` ターゲットは、ピンを指定しなくても
	// 内部で自動的に正しいSDA/SCLピン（SDA0/SCL0）を割り当ててくれます。
	i2c := machine.I2C0
	err := i2c.Configure(machine.I2CConfig{
		Frequency: 400e3, // 周波数だけ指定
	})
	if err != nil {
		println("I2C configured error:", err.Error())
	} else {
		println("I2C configured successfully")
	}

	// AXP192の初期化
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

	width, height := display.Size()

	display.FillScreen(black)

	display.FillRectangle(width/4, height/4, width/2, height/2, blue)

	drawFilledCircle(display, width/2, height/2, 30, green)

	tinyfont.WriteLine(display, &freemono.Regular9pt7b, 30, 40, "Hello M5Stack ...", green)
}

func drawFilledCircle(display *ili9341.Device, x, y, r int16, c color.RGBA) {
	for dy := int16(-r); dy <= r; dy++ {
		for dx := int16(-r); dx <= r; dx++ {
			if dx*dx+dy*dy <= r*r {
				display.SetPixel(x+dx, y+dy, c)
			}
		}
	}
}
