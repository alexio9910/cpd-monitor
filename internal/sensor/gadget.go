// Paquete sensor: sabe hablar por BLE con un Sensirion SHT4x Smart Gadget
// (el mismo perfil BLE lo usan también los gadgets SHT31/SHTC1 anteriores).
//
// El gadget expone tres características GATT que nos interesan:
//
//   - Humedad relativa:  UUID 00001235-b38d-4985-720e-0f993a68ee41  (float32)
//   - Temperatura:       UUID 00002235-b38d-4985-720e-0f993a68ee41  (float32)
//   - Batería:           UUID estándar Bluetooth 0x2A19             (uint8, %)
//
// Estos UUID son los que publica el propio firmware de Sensirion
// (github.com/Sensirion/SmartGadget-Firmware, servicios HumidityService y
// TemperatureService) y están confirmados de forma independiente por varios
// proyectos de la comunidad que han leído el valor directamente del
// dispositivo con gatttool/nRF Connect. Aun así, la recomendación del
// paso 1 del README (escanear el gadget con nRF Connect) sirve precisamente
// para que confirmes estos UUID contra TU unidad física antes de fiarte
// de ellos a ciegas: Sensirion no garantiza que no cambien en una revisión
// de firmware futura.
package sensor

import (
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	uuidServicioHumedad     = "00001235-b38d-4985-720e-0f993a68ee41"
	uuidServicioTemperatura = "00002235-b38d-4985-720e-0f993a68ee41"
)

// Lectura contiene una medición completa de un gadget.
type Lectura struct {
	TemperaturaC  float32
	HumedadRelPct float32

	// BateriaPct puede ser -1 si no se ha podido leer la batería
	// (algunas unidades más antiguas no exponen esta característica).
	BateriaPct int
}

// adaptador es el único adaptador Bluetooth del host. BlueZ (a través de
// D-Bus) no está pensado para atender varias operaciones de conexión GATT
// concurrentes sobre el mismo adaptador físico, así que TODAS las lecturas
// de TODOS los sensores pasan por aquí de una en una. Ver Leer().
var adaptador = bluetooth.DefaultAdapter

var adaptadorHabilitado = false

// Habilitar enciende la pila BLE del sistema. Se llama una sola vez al
// arrancar el colector (cmd/collector/main.go), nunca por cada lectura.
func Habilitar() error {
	if adaptadorHabilitado {
		return nil
	}
	if err := adaptador.Enable(); err != nil {
		return fmt.Errorf("no se pudo habilitar el adaptador Bluetooth del sistema: %w", err)
	}
	adaptadorHabilitado = true
	return nil
}

// Leer se conecta al gadget indicado por su MAC, lee humedad, temperatura
// y batería, y se desconecta. Es una operación "todo en uno" deliberadamente
// simple: para una lectura cada 60 segundos no compensa la complejidad de
// mantener conexiones BLE persistentes ni suscripciones a notificaciones.
func Leer(mac string, timeout time.Duration) (Lectura, error) {
	var vacio Lectura

	direccionMAC, err := bluetooth.ParseMAC(mac)
	if err != nil {
		return vacio, fmt.Errorf("la MAC %q no tiene un formato válido (usa AA:BB:CC:DD:EE:FF): %w", mac, err)
	}
	direccion := bluetooth.Address{MACAddress: bluetooth.MACAddress{MAC: direccionMAC}}

	tipoConexion := make(chan error, 1)
	var dispositivo bluetooth.Device

	go func() {
		var errConexion error
		dispositivo, errConexion = adaptador.Connect(direccion, bluetooth.ConnectionParams{})
		tipoConexion <- errConexion
	}()

	select {
	case err := <-tipoConexion:
		if err != nil {
			return vacio, fmt.Errorf("no se pudo conectar con el sensor %s: %w", mac, err)
		}
	case <-time.After(timeout):
		return vacio, fmt.Errorf("tiempo de espera agotado conectando con el sensor %s", mac)
	}
	defer dispositivo.Disconnect() //nolint:errcheck // best-effort al cerrar

	uuidHumedad, err := bluetooth.ParseUUID(uuidServicioHumedad)
	if err != nil {
		return vacio, fmt.Errorf("UUID de humedad inválido en el código: %w", err)
	}
	uuidTemperatura, err := bluetooth.ParseUUID(uuidServicioTemperatura)
	if err != nil {
		return vacio, fmt.Errorf("UUID de temperatura inválido en el código: %w", err)
	}
	uuidBateria := bluetooth.New16BitUUID(0x2A19)

	servicios, err := dispositivo.DiscoverServices([]bluetooth.UUID{uuidHumedad, uuidTemperatura, uuidBateria})
	if err != nil {
		return vacio, fmt.Errorf("no se pudieron descubrir los servicios BLE del sensor %s: %w", mac, err)
	}

	var lectura Lectura
	lectura.BateriaPct = -1
	huboHumedad, huboTemperatura := false, false

	for _, servicio := range servicios {
		caracteristicas, err := servicio.DiscoverCharacteristics(nil)
		if err != nil {
			continue
		}
		for _, c := range caracteristicas {
			switch c.UUID().String() {
			case uuidServicioHumedad:
				valor, err := leerFloat32(c)
				if err == nil {
					lectura.HumedadRelPct = valor
					huboHumedad = true
				}
			case uuidServicioTemperatura:
				valor, err := leerFloat32(c)
				if err == nil {
					lectura.TemperaturaC = valor
					huboTemperatura = true
				}
			case uuidBateria.String():
				buf := make([]byte, 4)
				n, err := c.Read(buf)
				if err == nil && n >= 1 {
					lectura.BateriaPct = int(buf[0])
				}
			}
		}
	}

	if !huboHumedad || !huboTemperatura {
		return vacio, fmt.Errorf("el sensor %s respondió pero no se pudo leer humedad y/o temperatura", mac)
	}

	return lectura, nil
}

// leerFloat32 decodifica una característica del gadget: los valores de
// humedad y temperatura se transmiten como IEEE-754 float32 little-endian.
func leerFloat32(c bluetooth.DeviceCharacteristic) (float32, error) {
	buf := make([]byte, 4)
	n, err := c.Read(buf)
	if err != nil {
		return 0, err
	}
	if n < 4 {
		return 0, fmt.Errorf("se esperaban 4 bytes y se recibieron %d", n)
	}
	bits := binary.LittleEndian.Uint32(buf)
	return math.Float32frombits(bits), nil
}
