// Package bmi270 implements a driver for the BMI270 6-axis IMU.
//
// Datasheet: https://www.bosch-sensortec.com/media/boschsensortec/downloads/datasheets/bst-bmi270-ds000.pdf
package bmi270

import (
	"errors"
	"time"

	"tinygo.org/x/drivers"
)

const (
	Address = 0x68

	CHIP_ID_REG         = 0x00
	ACC_DATA_REG        = 0x0C
	INTERNAL_STATUS_REG = 0x21
	ACC_CONF_REG        = 0x40
	ACC_RANGE_REG       = 0x41
	INIT_CTRL_REG       = 0x59
	INIT_ADDR_0_REG     = 0x5B
	INIT_ADDR_1_REG     = 0x5C
	INIT_DATA_REG       = 0x5E
	PWR_CONF_REG        = 0x7C
	PWR_CTRL_REG        = 0x7D
	CMD_REG             = 0x7E

	CHIP_ID_BMI270 = 0x24
)

type AccelRange uint8

const (
	ACCEL_2G  AccelRange = 0x00
	ACCEL_4G  AccelRange = 0x01
	ACCEL_8G  AccelRange = 0x02
	ACCEL_16G AccelRange = 0x03
)

type Device struct {
	bus        drivers.I2C
	Address    uint16
	accelRange AccelRange
	wbuf       [17]byte
	rbuf       [6]byte
}

type Config struct {
	AccelRange AccelRange
}

func DefaultConfig() Config {
	return Config{
		AccelRange: ACCEL_2G,
	}
}

var (
	errNotConnected = errors.New("bmi270: not connected")
	errInitFailed   = errors.New("bmi270: initialization failed")
)

func New(bus drivers.I2C) *Device {
	return &Device{
		bus:     bus,
		Address: Address,
	}
}

func (d *Device) Connected() bool {
	d.wbuf[0] = CHIP_ID_REG
	if err := d.bus.Tx(d.Address, d.wbuf[:1], d.rbuf[:1]); err != nil {
		return false
	}
	return d.rbuf[0] == CHIP_ID_BMI270
}

func (d *Device) Configure(cfg Config) error {
	if !d.Connected() {
		return errNotConnected
	}

	if cfg.AccelRange != 0 {
		d.accelRange = cfg.AccelRange
	} else {
		d.accelRange = ACCEL_2G
	}

	if err := d.writeRegister(CMD_REG, 0xB6); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	d.readRegister(CHIP_ID_REG, d.rbuf[:1])

	if err := d.writeRegister(PWR_CONF_REG, 0x00); err != nil {
		return err
	}
	time.Sleep(1 * time.Millisecond)

	if err := d.writeRegister(INIT_CTRL_REG, 0x00); err != nil {
		return err
	}
	time.Sleep(1 * time.Millisecond)

	chunkSize := 16
	for i := 0; i < len(configData); i += chunkSize {
		end := i + chunkSize
		if end > len(configData) {
			end = len(configData)
		}

		wordAddr := uint16(i / 2)
		addrLow := byte(wordAddr & 0x0F)
		addrHigh := byte(wordAddr >> 4)
		d.wbuf[0] = INIT_ADDR_0_REG
		d.wbuf[1] = addrLow
		d.wbuf[2] = addrHigh
		if err := d.bus.Tx(d.Address, d.wbuf[:3], nil); err != nil {
			return err
		}

		chunk := configData[i:end]
		d.wbuf[0] = INIT_DATA_REG
		copy(d.wbuf[1:], chunk)
		if err := d.bus.Tx(d.Address, d.wbuf[:1+len(chunk)], nil); err != nil {
			return err
		}
	}

	if err := d.writeRegister(INIT_CTRL_REG, 0x01); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	retries := 10
	for retries > 0 {
		d.readRegister(INTERNAL_STATUS_REG, d.rbuf[:1])
		if d.rbuf[0] == 0x01 {
			break
		}
		time.Sleep(50 * time.Millisecond)
		retries--
	}
	if retries <= 0 {
		return errInitFailed
	}

	if err := d.writeRegister(ACC_CONF_REG, 0xA8); err != nil {
		return err
	}

	var rangeVal byte
	switch d.accelRange {
	case ACCEL_2G:
		rangeVal = 0x00
	case ACCEL_4G:
		rangeVal = 0x01
	case ACCEL_8G:
		rangeVal = 0x02
	case ACCEL_16G:
		rangeVal = 0x03
	default:
		rangeVal = 0x00
	}
	if err := d.writeRegister(ACC_RANGE_REG, rangeVal); err != nil {
		return err
	}

	if err := d.writeRegister(PWR_CTRL_REG, 0x04); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	return nil
}

// ReadAcceleration returns the acceleration in µg (micro-gravity).
// When one of the axes is pointing straight down and the sensor
// is not moving, the returned value will be around 1000000.
func (d *Device) ReadAcceleration() (x, y, z int32, err error) {
	if err = d.readRegister(ACC_DATA_REG, d.rbuf[:6]); err != nil {
		return 0, 0, 0, err
	}

	rawX := int32(int16((uint16(d.rbuf[1]) << 8) | uint16(d.rbuf[0])))
	rawY := int32(int16((uint16(d.rbuf[3]) << 8) | uint16(d.rbuf[2])))
	rawZ := int32(int16((uint16(d.rbuf[5]) << 8) | uint16(d.rbuf[4])))

	k := int32(61)
	switch d.accelRange {
	case ACCEL_4G:
		k = 122
	case ACCEL_8G:
		k = 244
	case ACCEL_16G:
		k = 488
	}

	x = rawX * k
	y = rawY * k
	z = rawZ * k
	return
}

func (d *Device) readRegister(reg byte, data []byte) error {
	d.wbuf[0] = reg
	return d.bus.Tx(d.Address, d.wbuf[:1], data)
}

func (d *Device) writeRegister(reg, val byte) error {
	d.wbuf[0] = reg
	d.wbuf[1] = val
	return d.bus.Tx(d.Address, d.wbuf[:2], nil)
}