package i2csoft

import (
	"errors"
	"machine"
	"time"

	"tinygo.org/x/drivers/delay"
)

type I2C struct {
	scl      machine.Pin
	sda      machine.Pin
	nack     bool
	baudrate uint32
}

type I2CConfig struct {
	Frequency uint32
	SCL       machine.Pin
	SDA       machine.Pin
}

var (
	errSI2CAckExpected = errors.New("I2C error: expected ACK not NACK")
)

func New(sclPin, sdaPin machine.Pin) *I2C {
	return &I2C{
		scl:      sclPin,
		sda:      sdaPin,
		baudrate: 100e3,
	}
}

func (i2c *I2C) Configure(config I2CConfig) error {
	if config.Frequency != 0 {
		i2c.SetBaudRate(config.Frequency)
	}

	if config.SCL != config.SDA {
		i2c.scl = config.SCL
		i2c.sda = config.SDA
	}

	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.sda.High()
	i2c.scl.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.scl.High()

	return nil
}

func (i2c *I2C) SetBaudRate(br uint32) {
	i2c.baudrate = br
}

func (i2c *I2C) Tx(addr uint16, w, r []byte) error {
	i2c.nack = false
	if len(w) != 0 {
		i2c.sendAddress(addr, true)

		if i2c.nack {
			i2c.signalStop()
			return errSI2CAckExpected
		}

		for _, b := range w {
			i2c.writeByte(b)
		}

		if len(r) == 0 {
			i2c.signalStop()
		}
	}
	if len(r) != 0 {
		i2c.sendAddress(addr, false)

		if i2c.nack {
			i2c.signalStop()
			return errSI2CAckExpected
		}

		r[0] = i2c.readByte()
		for i := 1; i < len(r); i++ {
			i2c.signalRead()
			r[i] = i2c.readByte()
		}

		i2c.sendNack()
		i2c.signalStop()
	}

	return nil
}

func (i2c *I2C) writeByte(data byte) {
	i2c.scl.Low()
	i2c.sda.High()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.wait()

	for i := 0; i < 8; i++ {
		i2c.scl.Low()
		if ((data >> (7 - i)) & 1) == 1 {
			i2c.sda.High()
		} else {
			i2c.sda.Low()
		}
		i2c.wait()
		i2c.wait()
		i2c.scl.High()
		i2c.wait()
		i2c.wait()
	}

	i2c.scl.Low()
	i2c.wait()
	i2c.wait()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinInput})
	i2c.scl.High()
	i2c.wait()

	i2c.nack = i2c.sda.Get()

	i2c.wait()
}

func (i2c *I2C) sendAddress(address uint16, write bool) {
	data := (address << 1)
	if !write {
		data |= 1
	}

	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.sda.High()
	i2c.scl.High()
	i2c.wait()
	i2c.wait()
	i2c.sda.Low()
	i2c.wait()
	i2c.wait()

	for i := 0; i < 8; i++ {
		i2c.scl.Low()
		if ((data >> (7 - i)) & 1) == 1 {
			i2c.sda.High()
		} else {
			i2c.sda.Low()
		}
		i2c.wait()
		i2c.wait()
		i2c.scl.High()
		i2c.wait()
		i2c.wait()
	}

	i2c.scl.Low()
	i2c.wait()
	i2c.wait()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinInput})
	i2c.scl.High()
	i2c.wait()

	i2c.nack = i2c.sda.Get()

	i2c.wait()
}

func (i2c *I2C) signalStart() {
	i2c.scl.High()
	i2c.wait()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.sda.Low()
	i2c.wait()
	i2c.wait()
}

func (i2c *I2C) signalStop() {
	i2c.scl.Low()
	i2c.sda.Low()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.wait()
	i2c.wait()
	i2c.scl.High()
	i2c.wait()
	i2c.wait()
	i2c.sda.High()
	i2c.wait()
	i2c.wait()
}

func (i2c *I2C) signalRead() {
	i2c.wait()
	i2c.wait()
	i2c.scl.Low()
	i2c.sda.Low()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.wait()
	i2c.wait()
	i2c.scl.High()
	i2c.wait()
	i2c.wait()
}

func (i2c *I2C) readByte() byte {
	var data byte
	for i := 0; i < 8; i++ {
		i2c.scl.Low()
		i2c.sda.Configure(machine.PinConfig{Mode: machine.PinInput})
		i2c.wait()
		i2c.wait()
		i2c.scl.High()
		if i2c.sda.Get() {
			data |= 1 << (7 - i)
		}
		i2c.wait()
		i2c.wait()
	}
	return data
}

func (i2c *I2C) sendNack() {
	i2c.wait()
	i2c.wait()
	i2c.scl.Low()
	i2c.sda.High()
	i2c.sda.Configure(machine.PinConfig{Mode: machine.PinOutput})
	i2c.wait()
	i2c.wait()
	i2c.scl.High()
	i2c.wait()
	i2c.wait()
}

func (i2c *I2C) WriteRegister(address uint8, register uint8, data []byte) error {
	buf := make([]uint8, len(data)+1)
	buf[0] = register
	copy(buf[1:], data)
	return i2c.Tx(uint16(address), buf, nil)
}

func (i2c *I2C) ReadRegister(address uint8, register uint8, data []byte) error {
	return i2c.Tx(uint16(address), []byte{register}, data)
}

func (i2c *I2C) wait() {
	delay.Sleep(50 * time.Microsecond)
}
