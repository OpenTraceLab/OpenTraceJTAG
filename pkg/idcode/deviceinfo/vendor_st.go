package deviceinfo

// STMicroelectronics device entries
func init() {
	const stm = 0x020 // STMicroelectronics JEP106 code

	// STM32F1 series
	register(key{ManufacturerCode: stm, PartNumber: 0x410}, DeviceInfo{
		Name:            "STM32F10x (Medium-density)",
		Family:          "STM32F1",
		Description:     "ARM Cortex-M3 MCU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M3",
		IsMCU:           true,
		IRLength:        4,
	})

	register(key{ManufacturerCode: stm, PartNumber: 0x412}, DeviceInfo{
		Name:            "STM32F10x (Low-density)",
		Family:          "STM32F1",
		Description:     "ARM Cortex-M3 MCU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M3",
		IsMCU:           true,
		IRLength:        4,
	})

	register(key{ManufacturerCode: stm, PartNumber: 0x414}, DeviceInfo{
		Name:            "STM32F10x (High-density)",
		Family:          "STM32F1",
		Description:     "ARM Cortex-M3 MCU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M3",
		IsMCU:           true,
		IRLength:        4,
	})

	// STM32F4 series
	register(key{ManufacturerCode: stm, PartNumber: 0x413}, DeviceInfo{
		Name:            "STM32F40x/41x",
		Family:          "STM32F4",
		Description:     "ARM Cortex-M4 MCU with FPU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M4",
		IsMCU:           true,
		IRLength:        4,
		DatasheetURL:    "https://www.st.com/resource/en/datasheet/stm32f407vg.pdf",
	})

	register(key{ManufacturerCode: stm, PartNumber: 0x419}, DeviceInfo{
		Name:            "STM32F42x/43x",
		Family:          "STM32F4",
		Description:     "ARM Cortex-M4 MCU with FPU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M4",
		IsMCU:           true,
		IRLength:        4,
	})

	// STM32F3 series
	register(key{ManufacturerCode: stm, PartNumber: 0x422}, DeviceInfo{
		Name:            "STM32F30x/31x",
		Family:          "STM32F3",
		Description:     "ARM Cortex-M4 MCU with FPU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M4",
		IsMCU:           true,
		IRLength:        4,
	})

	// STM32F7 series
	register(key{ManufacturerCode: stm, PartNumber: 0x449}, DeviceInfo{
		Name:            "STM32F74x/75x",
		Family:          "STM32F7",
		Description:     "ARM Cortex-M7 MCU with FPU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M7",
		IsMCU:           true,
		IRLength:        4,
	})

	// STM32H7 series
	register(key{ManufacturerCode: stm, PartNumber: 0x450}, DeviceInfo{
		Name:            "STM32H74x/75x",
		Family:          "STM32H7",
		Description:     "ARM Cortex-M7 MCU with FPU",
		HasBoundaryScan: true,
		HasARMCore:      true,
		ARMCore:         "Cortex-M7",
		IsMCU:           true,
		IRLength:        4,
	})
}
