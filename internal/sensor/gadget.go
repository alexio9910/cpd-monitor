// Paquete sensor: sabe hablar por BLE con sensores Sensirion SHT4x Smart
// Gadget, comunicándose DIRECTAMENTE con BlueZ vía D-Bus.
//
// DECISIÓN DE DISEÑO — conexión BLE persistente por sensor: la primera
// versión conectaba, leía y desconectaba en cada ciclo. En la práctica
// eso resultó ser la causa de fallos intermitentes: al desconectar,
// BlueZ olvida el árbol GATT resuelto, así que el siguiente ciclo tiene
// que redescubrir ~15 servicios desde cero, lo cual a veces supera el
// timeout. Ahora cada sensor mantiene su conexión abierta mientras el
// colector esté vivo; solo se reconecta si una lectura falla (indicio de
// que la conexión se cortó de verdad). Ver docs/ARQUITECTURA.md.
//
// También se descartó la librería de abstracción tinygo.org/x/bluetooth:
// tiene un problema de compatibilidad conocido y sin resolver con
// versiones recientes de BlueZ (ver
// https://github.com/tinygo-org/bluetooth/issues/118). Hablar con BlueZ
// directamente por D-Bus —el mismo mecanismo que usa `bluetoothctl`— es
// más código pero mucho más fiable en la práctica.
//
// El gadget expone tres características GATT que nos interesan:
//
//   - Humedad relativa:  UUID 00001235-b38d-4985-720e-0f993a68ee41  (float32)
//   - Temperatura:       UUID 00002235-b38d-4985-720e-0f993a68ee41  (float32)
//   - Batería:           UUID estándar Bluetooth 0x2A19             (uint8, %)
package sensor

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	uuidServicioHumedad     = "00001235-b38d-4985-720e-0f993a68ee41"
	uuidServicioTemperatura = "00002235-b38d-4985-720e-0f993a68ee41"
	uuidBateria             = "00002a19-0000-1000-8000-00805f9b34fb"

	adaptadorBluez = "hci0"

	intentosConexion    = 3
	esperaEntreIntentos = 1500 * time.Millisecond
)

// Lectura contiene una medición completa de un gadget.
type Lectura struct {
	TemperaturaC  float32
	HumedadRelPct float32
	BateriaPct    int // -1 si no se pudo leer
}

// conexionSensor guarda el estado persistente de la conexión BLE a un
// sensor: si está conectada ahora mismo, y las rutas D-Bus de sus
// características GATT ya localizadas (para no buscarlas en cada lectura).
type conexionSensor struct {
	rutaDispositivo dbus.ObjectPath
	rutaHumedad     dbus.ObjectPath
	rutaTemperatura dbus.ObjectPath
	rutaBateria     dbus.ObjectPath
	conectado       bool
}

var (
	conn       *dbus.Conn
	mu         sync.Mutex
	conexiones = map[string]*conexionSensor{}
)

// Habilitar abre la conexión al bus D-Bus del sistema.
func Habilitar() error {
	if conn != nil {
		return nil
	}
	c, err := dbus.ConnectSystemBus()
	if err != nil {
		return fmt.Errorf("no se pudo conectar al bus D-Bus del sistema: %w", err)
	}
	conn = c
	return nil
}

func rutaDispositivo(mac string) dbus.ObjectPath {
	addr := strings.ReplaceAll(mac, ":", "_")
	return dbus.ObjectPath(fmt.Sprintf("/org/bluez/%s/dev_%s", adaptadorBluez, addr))
}

func objetosGestionados() (map[dbus.ObjectPath]map[string]map[string]dbus.Variant, error) {
	raiz := conn.Object("org.bluez", dbus.ObjectPath("/"))
	var gestionados map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if err := raiz.Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&gestionados); err != nil {
		return nil, err
	}
	return gestionados, nil
}

func objetoExiste(ruta dbus.ObjectPath) bool {
	gestionados, err := objetosGestionados()
	if err != nil {
		return false
	}
	_, ok := gestionados[ruta]
	return ok
}

// asegurarDispositivoConocido comprueba si BlueZ ya conoce el dispositivo;
// si no, lanza un descubrimiento breve (igual que `bluetoothctl scan on`).
// Imprescindible tras un reinicio del host, cuando BlueZ no tiene memoria
// de ningún sensor todavía.
func asegurarDispositivoConocido(ruta dbus.ObjectPath, timeout time.Duration) error {
	if objetoExiste(ruta) {
		return nil
	}
	adaptador := conn.Object("org.bluez", dbus.ObjectPath("/org/bluez/"+adaptadorBluez))
	if call := adaptador.Call("org.bluez.Adapter1.StartDiscovery", 0); call.Err != nil {
		return fmt.Errorf("no se pudo iniciar el escaneo BLE: %w", call.Err)
	}
	defer adaptador.Call("org.bluez.Adapter1.StopDiscovery", 0)

	limite := time.Now().Add(timeout)
	for time.Now().Before(limite) {
		if objetoExiste(ruta) {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("no se detectó el dispositivo tras %s de escaneo", timeout)
}

// conectar reutiliza la conexión BLE persistente al sensor si ya está
// viva; si no, la establece desde cero: conecta, espera a que se
// resuelvan los servicios GATT, y localiza las rutas D-Bus de sus tres
// características de interés.
func conectar(mac string, timeout time.Duration) (*conexionSensor, error) {
	mu.Lock()
	c, existe := conexiones[mac]
	mu.Unlock()
	if existe && c.conectado {
		return c, nil
	}

	ruta := rutaDispositivo(mac)
	if err := asegurarDispositivoConocido(ruta, timeout); err != nil {
		return nil, fmt.Errorf("no se pudo localizar el sensor %s: %w", mac, err)
	}

	dispositivo := conn.Object("org.bluez", ruta)

	var ultimoErr error
	conectadoOK := false
	for intento := 1; intento <= intentosConexion; intento++ {
		if call := dispositivo.Call("org.bluez.Device1.Connect", 0); call.Err == nil {
			conectadoOK = true
			break
		} else {
			ultimoErr = call.Err
		}
		if intento < intentosConexion {
			time.Sleep(esperaEntreIntentos)
		}
	}
	if !conectadoOK {
		return nil, fmt.Errorf("no se pudo conectar con el sensor %s: %w", mac, ultimoErr)
	}

	limite := time.Now().Add(timeout)
	resuelto := false
	for time.Now().Before(limite) {
		if variant, err := dispositivo.GetProperty("org.bluez.Device1.ServicesResolved"); err == nil {
			if b, ok := variant.Value().(bool); ok && b {
				resuelto = true
				break
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	if !resuelto {
		dispositivo.Call("org.bluez.Device1.Disconnect", 0)
		return nil, fmt.Errorf("tiempo de espera agotado resolviendo servicios BLE del sensor %s", mac)
	}

	gestionados, err := objetosGestionados()
	if err != nil {
		dispositivo.Call("org.bluez.Device1.Disconnect", 0)
		return nil, fmt.Errorf("no se pudieron listar los objetos BLE del sensor %s: %w", mac, err)
	}

	nueva := &conexionSensor{rutaDispositivo: ruta}
	for objPath, interfaces := range gestionados {
		if !strings.HasPrefix(string(objPath), string(ruta)+"/") {
			continue
		}
		propiedades, ok := interfaces["org.bluez.GattCharacteristic1"]
		if !ok {
			continue
		}
		uuidVariant, ok := propiedades["UUID"]
		if !ok {
			continue
		}
		switch strings.ToLower(fmt.Sprint(uuidVariant.Value())) {
		case uuidServicioHumedad:
			nueva.rutaHumedad = objPath
		case uuidServicioTemperatura:
			nueva.rutaTemperatura = objPath
		case uuidBateria:
			nueva.rutaBateria = objPath
		}
	}

	if nueva.rutaHumedad == "" || nueva.rutaTemperatura == "" {
		dispositivo.Call("org.bluez.Device1.Disconnect", 0)
		return nil, fmt.Errorf("el sensor %s no expone las características de humedad/temperatura esperadas", mac)
	}

	nueva.conectado = true
	mu.Lock()
	conexiones[mac] = nueva
	mu.Unlock()
	return nueva, nil
}

// Leer lee humedad, temperatura y batería del sensor indicado, reutilizando
// la conexión BLE persistente (o creándola si hace falta).
func Leer(mac string, timeout time.Duration) (Lectura, error) {
	var vacio Lectura
	if conn == nil {
		return vacio, fmt.Errorf("el adaptador Bluetooth no está habilitado")
	}

	c, err := conectar(mac, timeout)
	if err != nil {
		return vacio, err
	}

	temp, errTemp := leerFloat32(c.rutaTemperatura)
	hum, errHum := leerFloat32(c.rutaHumedad)

	if errTemp != nil || errHum != nil {
		// La lectura falló con una conexión que creíamos viva: lo más
		// probable es que se haya cortado de verdad (sensor fuera de
		// alcance, sin batería...). La marcamos como caída para que el
		// siguiente ciclo la reestablezca desde cero.
		mu.Lock()
		c.conectado = false
		mu.Unlock()
		if errTemp != nil {
			return vacio, fmt.Errorf("no se pudo leer la temperatura del sensor %s: %w", mac, errTemp)
		}
		return vacio, fmt.Errorf("no se pudo leer la humedad del sensor %s: %w", mac, errHum)
	}

	lectura := Lectura{TemperaturaC: temp, HumedadRelPct: hum, BateriaPct: -1}
	if c.rutaBateria != "" {
		if buf, err := leerBytes(c.rutaBateria); err == nil && len(buf) >= 1 {
			lectura.BateriaPct = int(buf[0])
		}
	}
	return lectura, nil
}

func leerBytes(ruta dbus.ObjectPath) ([]byte, error) {
	obj := conn.Object("org.bluez", ruta)
	var buf []byte
	call := obj.Call("org.bluez.GattCharacteristic1.ReadValue", 0, map[string]dbus.Variant{})
	if call.Err != nil {
		return nil, call.Err
	}
	if err := call.Store(&buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func leerFloat32(ruta dbus.ObjectPath) (float32, error) {
	buf, err := leerBytes(ruta)
	if err != nil {
		return 0, err
	}
	if len(buf) < 4 {
		return 0, fmt.Errorf("se esperaban 4 bytes y se recibieron %d", len(buf))
	}
	bits := binary.LittleEndian.Uint32(buf)
	return math.Float32frombits(bits), nil
}

// DesconectarTodos cierra todas las conexiones BLE persistentes abiertas.
// Se llama al parar el colector (Ctrl+C / systemctl stop), para dejar
// BlueZ en un estado limpio.
func DesconectarTodos() {
	mu.Lock()
	defer mu.Unlock()
	for _, c := range conexiones {
		if c.conectado {
			dispositivo := conn.Object("org.bluez", c.rutaDispositivo)
			dispositivo.Call("org.bluez.Device1.Disconnect", 0)
			c.conectado = false
		}
	}
}
