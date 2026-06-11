package main

import (
	"machine"
	"time"

	"tinygo-sample/gravity/bmi270"
)

func main() {
	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{
		Frequency: 400e3,
		SDA:       machine.SDA0_PIN,
		SCL:       machine.SCL0_PIN,
	})

	imu := bmi270.NewI2C(i2c, bmi270.Address)
	if err := imu.Configure(bmi270.DefaultConfig()); err != nil {
		println("BMI270 error:", err.Error())
		return
	}
	println("BMI270 initialized")

	for {
		ax, ay, az, err := imu.ReadAcceleration()
		if err != nil {
			println("accel error:", err.Error())
			time.Sleep(100 * time.Millisecond)
			continue
		}
		gx, gy, gz, err := imu.ReadRotation()
		if err != nil {
			println("gyro error:", err.Error())
			time.Sleep(100 * time.Millisecond)
			continue
		}

		print("acc(mg): ", ax/1000, " ", ay/1000, " ", az/1000)
		print("  gyr(dps): ", gx/1000000, " ", gy/1000000, " ", gz/1000000)
		println()

		time.Sleep(100 * time.Millisecond)
	}
}