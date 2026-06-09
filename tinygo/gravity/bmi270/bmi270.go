package bmi270

import (
	"machine"
	"time"
)

const (
	Address = 0x68

	CHIP_ID_REG         = 0x00
	ACC_DATA_REG        = 0x0C // 修正: 0x12はジャイロ
	INTERNAL_STATUS_REG = 0x21 // 修正: 0x2AはFIFO領域
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

type Device struct {
	i2c *machine.I2C
}

func New(i2c *machine.I2C) *Device {
	return &Device{i2c: i2c}
}

func (d *Device) writeReg(reg byte, val byte) error {
	return d.i2c.Tx(Address, []byte{reg, val}, nil)
}

func (d *Device) readReg(reg byte) (byte, error) {
	buf := []byte{0}
	err := d.i2c.Tx(Address, []byte{reg}, buf)
	return buf[0], err
}

func (d *Device) Init() error {
	chipID, err := d.readReg(CHIP_ID_REG)
	if err != nil {
		return err
	}
	println("Chip ID:", chipID)

	if chipID != CHIP_ID_BMI270 {
		return nil
	}

	// ソフトリセット
	if err := d.writeReg(CMD_REG, 0xB6); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	// ダミーリード
	chipID, _ = d.readReg(CHIP_ID_REG)
	println("Chip ID after reset:", chipID)

	// Advanced Power Saveを無効化
	if err := d.writeReg(PWR_CONF_REG, 0x00); err != nil {
		return err
	}
	time.Sleep(1 * time.Millisecond)

	// INIT_CTRL = 0 (prepare for config load)
	if err := d.writeReg(INIT_CTRL_REG, 0x00); err != nil {
		return err
	}
	time.Sleep(1 * time.Millisecond)

	// Config dataをチャンクで書き込み（アドレスは1トランザクションで設定）
	chunkSize := 16
	println("Transferring", len(configData), "bytes")

	for i := 0; i < len(configData); i += chunkSize {
		end := i + chunkSize
		if end > len(configData) {
			end = len(configData)
		}

		wordAddr := uint16(i / 2)
		addrLow := byte(wordAddr & 0x0F)
		addrHigh := byte(wordAddr >> 4)
		// INIT_ADDR_0 + INIT_ADDR_1 をまとめて書き込み（auto-increment）
		if err := d.i2c.Tx(Address, []byte{INIT_ADDR_0_REG, addrLow, addrHigh}, nil); err != nil {
			return err
		}

		chunk := configData[i:end]
		buf := make([]byte, len(chunk)+1)
		buf[0] = INIT_DATA_REG
		copy(buf[1:], chunk)
		if err := d.i2c.Tx(Address, buf, nil); err != nil {
			println("Write failed at", i)
			return err
		}
	}
	println("Config transfer complete")

	// INIT_CTRL = 1 (start initialization)
	if err := d.writeReg(INIT_CTRL_REG, 0x01); err != nil {
		return err
	}
	time.Sleep(200 * time.Millisecond)

	// INTERNAL_STATUSを確認（リトライ付き）
	for i := 0; i < 10; i++ {
		intStatus, _ := d.readReg(INTERNAL_STATUS_REG)
		println("INTERNAL_STATUS:", intStatus)
		if intStatus == 0x01 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// ACC_CONF: ODR=100Hz, BWP=normal, filter_perf=1
	if err := d.writeReg(ACC_CONF_REG, 0xA8); err != nil {
		return err
	}

	// ACC_RANGE: +/-2g
	if err := d.writeReg(ACC_RANGE_REG, 0x00); err != nil {
		return err
	}

	// PWR_CTRL: ACC_EN=1
	if err := d.writeReg(PWR_CTRL_REG, 0x04); err != nil {
		return err
	}
	time.Sleep(50 * time.Millisecond)

	return nil
}

func (d *Device) ReadAcceleration() (x, y, z float64, err error) {
	data := make([]byte, 6)
	if err = d.i2c.Tx(Address, []byte{ACC_DATA_REG}, data); err != nil {
		return 0, 0, 0, err
	}

	rawX := int32(int16((uint16(data[1]) << 8) | uint16(data[0])))
	rawY := int32(int16((uint16(data[3]) << 8) | uint16(data[2])))
	rawZ := int32(int16((uint16(data[5]) << 8) | uint16(data[4])))

	x = float64(rawX) / 16384.0
	y = float64(rawY) / 16384.0
	z = float64(rawZ) / 16384.0

	return x, y, z, nil
}
