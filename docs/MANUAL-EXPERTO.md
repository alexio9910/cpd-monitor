# Manual de experto — CPD Monitor

Este documento es la referencia única para **entender, usar, diagnosticar y
modificar** el sistema de monitorización del CPD, sin necesitar
conocimientos de programación. No hace falta leerlo entero: busca la
sección que necesites.

> 📄 Si necesitas el detalle técnico de por qué se construyó así (para
> desarrolladores), consulta `docs/ARQUITECTURA.md`. Este manual es la
> versión práctica, orientada a "¿qué hago si...?".

---

## Índice

1. [Cómo funciona el sistema, en dos frases](#1-cómo-funciona-el-sistema-en-dos-frases)
2. [Cómo leer el dashboard de Grafana](#2-cómo-leer-el-dashboard-de-grafana)
3. [Qué hacer cuando llega un email de alerta](#3-qué-hacer-cuando-llega-un-email-de-alerta)
4. [Cambiar quién recibe las alertas por email](#4-cambiar-quién-recibe-las-alertas-por-email)
5. [Cambiar los rangos normales de temperatura/humedad](#5-cambiar-los-rangos-normales-de-temperaturahumedad)
6. [Gestionar usuarios de Grafana (dar/quitar acceso)](#6-gestionar-usuarios-de-grafana-darquitar-acceso)
7. [Añadir o quitar un sensor](#7-añadir-o-quitar-un-sensor)
8. [Mantenimiento básico (reiniciar, ver logs)](#8-mantenimiento-básico-reiniciar-ver-logs)
9. [Algo no funciona — diagnóstico rápido](#9-algo-no-funciona--diagnóstico-rápido)
10. [Dónde está todo (referencia rápida)](#10-dónde-está-todo-referencia-rápida)
11. [Configurar una IP fija para la Raspberry Pi](#11-configurar-una-ip-fija-para-la-raspberry-pi)
12. [Glosario](#12-glosario)

---

## 1. Cómo funciona el sistema, en dos frases

Dos sensores Bluetooth miden temperatura y humedad del CPD. Un programa en
la Raspberry Pi los lee cada minuto y guarda los datos; Grafana los dibuja
en gráficas y avisa por email si algo se sale de rango.

```
Sensor Bluetooth → Raspberry Pi (colector) → Base de datos → Grafana → Tú
```

Todo corre **dentro de la Raspberry Pi** (IP `IP_DE_TU_RASPBERRY` en tu red
local) — no depende de ningún servicio externo ni de internet, salvo para
mandar los emails de alerta.

---

## 2. Cómo leer el dashboard de Grafana

Entra en `http://IP_DE_TU_RASPBERRY:3000` desde cualquier navegador de la red
local. Usuarios disponibles:

| Usuario | Para qué sirve |
|---|---|
| `admin` | Gestionar el sistema (editar, borrar, configurar). Contraseña en `.env` → `GRAFANA_ADMIN_PASSWORD`. |
| `visor` | Solo ver las gráficas, sin poder tocar nada. Para dar acceso a quien solo necesite consultar. Contraseña en `.env` → `GRAFANA_VIEWER_PASSWORD`. |

Abre el dashboard **"CPD - Temperatura y Humedad"**. Los paneles:

- **Temperatura (°C) / Humedad relativa (%H)** — gráficas de las últimas
  24h. Si hay dos sensores, cada uno aparece como una línea de color
  distinto (mira la leyenda debajo de cada gráfica).
- **Temperatura actual / Humedad actual / Batería — Sensor 1** — el
  último valor de ese sensor concreto, con colores: **verde** = normal,
  **naranja** = acercándose al límite, **rojo** = fuera de rango.
- Si añades un segundo sensor, tendrá sus propios tres paneles
  "— Sensor 2" (ver sección 7).

---

## 3. Qué hacer cuando llega un email de alerta

El asunto empieza siempre con:

- **🔴 ALERTA** → algo está fuera de rango ahora mismo.
- **🟢 RESUELTA** → ya ha vuelto a rango normal por sí solo. No requiere
  ninguna acción, es solo confirmación de que se arregló.

Cuando llegue un **🔴 ALERTA**:

1. Mira el cuerpo del correo: te dice el sensor, la ubicación, el valor
   exacto y desde cuándo.
2. Entra en el dashboard (enlace incluido en el propio correo) para ver
   si es un pico puntual o una tendencia sostenida.
3. Si puedes, revisa físicamente el CPD (¿puerta abierta, aire
   acondicionado apagado, algo bloqueando la ventilación?).
4. Si en unos minutos no llega el **🟢 RESUELTA**, el problema sigue
   activo — actúa según lo que hayas visto en el paso 3, o avisa a
   alguien que pueda intervenir físicamente en el CPD.

**Si llega una alerta sin ningún valor** (o el asunto menciona
"sin datos"): no es un problema de temperatura, es que **el propio
colector ha dejado de mandar datos** — ver sección 9, "No llegan datos al
dashboard".

---

## 4. Cambiar quién recibe las alertas por email

Las direcciones **no** se editan en el YAML de Grafana directamente (ahí
solo hay una referencia a una variable, no las direcciones en sí) — se
cambian en tu fichero de secretos local:

```bash
ssh usuario@IP_DE_TU_RASPBERRY
cd ~/cpd-monitor
nano .env
```

Busca la línea `GRAFANA_ALERT_EMAILS=` y añade, quita o cambia
direcciones ahí, separadas por `;`:

```
GRAFANA_ALERT_EMAILS="persona1@tuempresa.com;persona2@tuempresa.com"
```

Guarda (`Ctrl+O`, `Enter`, `Ctrl+X`) y aplica el cambio:

```bash
docker compose up -d --force-recreate grafana
```

Prueba que llega bien: en Grafana → **Alerting → Contact points →
`email-cpd` → Test**.

> `.env` nunca se sube a GitHub — este cambio es puramente local en el
> host, no requiere ningún commit.

---

## 5. Cambiar los rangos normales de temperatura/humedad

Esto se edita desde la propia interfaz de Grafana, no hace falta tocar
ficheros:

1. Entra como `admin` → **Alerting → Alert rules**.
2. Abre la regla que quieras cambiar ("Temperatura CPD fuera de rango" o
   "Humedad CPD fuera de rango").
3. En la sección de condiciones (Threshold), cambia los números `18` y
   `27` (temperatura) o `40` y `60` (humedad) por los que necesites.
4. Baja hasta el final y pulsa **"Save rule and exit"**.

Los colores del dashboard (verde/naranja/rojo) son independientes y viven
en `grafana/dashboards/cpd-temp-humedad.json` — si quieres que coincidan
exactamente con los nuevos rangos, pide ayuda técnica para ese fichero en
concreto (es más delicado de editar a mano).

---

## 6. Gestionar usuarios de Grafana (dar/quitar acceso)

**Dar acceso de solo ver a alguien nuevo** — como `admin`, en Grafana:

1. **Administration → Users and access → Users → New user**.
2. Rellena nombre, usuario, contraseña.
3. Una vez creado, entra en ese usuario y ponle el rol **Viewer** (no
   Editor ni Admin, salvo que de verdad necesite poder editar).

**Cambiar una contraseña** (incluida la de `admin`): mismo menú,
selecciona el usuario → **Change password**.

**Quitar acceso a alguien**: mismo menú → selecciona el usuario →
**Delete user**.

---

## 7. Añadir o quitar un sensor

Procedimiento completo cuando llegue el segundo sensor físico (o
cualquier sensor adicional en el futuro). No hace falta recompilar nada.

### 7.1 — Encuentra la MAC del sensor nuevo

Con el sensor encendido y cerca de la Raspberry Pi:

```bash
ssh cpd@IP_DE_TU_RASPBERRY
bluetoothctl
scan on
```

Espera a ver su nombre (algo como `SHT40 Gadget`) y anota la dirección
tipo `AA:BB:CC:DD:EE:FF`. Luego:

```
scan off
exit
```

### 7.2 — Añádelo a la configuración

```bash
cd ~/cpd-monitor
nano config.yaml
```

Añade un bloque nuevo bajo `sensores:` (respeta la sangría exacta):

```yaml
  - id: "sensor2"
    mac: "AA:BB:CC:DD:EE:FF"
    ubicacion: "Sensor 2"
```

Guarda y despliega:

```bash
sudo cp config.yaml /opt/cpd-monitor/config.yaml
sudo chown cpdmonitor:cpdmonitor /opt/cpd-monitor/config.yaml
sudo systemctl restart cpd-monitor
journalctl -u cpd-monitor -f
```

Deberías ver una línea por sensor cada minuto. `Ctrl+C` para salir.

### 7.3 — Añade sus paneles de "valor actual" en Grafana

Las gráficas grandes (Temperatura, Humedad) **ya muestran el sensor
nuevo solas**, sin tocar nada. Los tres paneles pequeños de "valor
actual" (Temperatura, Humedad, Batería) sí hay que crearlos — con un
script ya preparado para ello, en vez de copiarlos a mano en la
interfaz:

````bash
cd ~/cpd-monitor
python3 scripts/anadir_sensor_dashboard.py sensor2 "Sensor 2"
docker compose restart grafana
````

El primer argumento es el mismo `id` que pusiste en `config.yaml`
(sección 7.2); el segundo, el texto que quieres que aparezca en el
título de los paneles. El script duplica los tres paneles de plantilla
(los de `sensor1`), les cambia el filtro y el título, y los coloca
automáticamente en una fila nueva debajo de los existentes — sirve
igual para un tercer sensor, un cuarto, etc. Si lo ejecutas dos veces
para el mismo sensor, detecta que ya existen y no hace nada (no
duplica).

### 7.4 — Las alertas no requieren ningún cambio

Las reglas de alerta ya cubren automáticamente cualquier sensor que
exista — no hay que crear reglas nuevas.

---

## 8. Mantenimiento básico (reiniciar, ver logs)

Todo esto es seguro de ejecutar, no borra datos:

| Quiero... | Comando (conectado por SSH a la Pi) |
|---|---|
| Ver qué está leyendo el colector ahora mismo | `journalctl -u cpd-monitor -f` (sal con `Ctrl+C`) |
| Reiniciar el colector | `sudo systemctl restart cpd-monitor` |
| Ver si el colector está bien | `sudo systemctl status cpd-monitor` |
| Reiniciar Grafana | `docker compose restart grafana` (dentro de `~/cpd-monitor`) |
| Reiniciar InfluxDB (la base de datos) | `docker compose restart influxdb` |
| Reiniciar TODO (tras un corte de luz, etc.) | Simplemente enciende la Raspberry Pi — todo arranca solo |
| Guardar un cambio de configuración en GitHub | `git add -A && git commit -m "describe aquí qué cambiaste"` y luego `git push` |

---

## 9. Algo no funciona — diagnóstico rápido

| Síntoma | Probablemente | Qué hacer |
|---|---|---|
| No veo datos nuevos en el dashboard | El colector no está leyendo el sensor | `journalctl -u cpd-monitor -f` y mira el último error. Si dice "no se pudo conectar", acércate al sensor o revisa su batería. |
| Me llega una alerta de "sin datos" | El colector está parado o la Pi tiene un problema | `sudo systemctl status cpd-monitor`. Si no está "active (running)", `sudo systemctl restart cpd-monitor`. |
| No puedo entrar en Grafana | El contenedor está caído, o la Pi está apagada | Conéctate por SSH y prueba `docker compose ps`. Si `cpd-grafana` no aparece "Up", `docker compose up -d`. |
| No me llegan alertas por email pero el dashboard sí tiene datos | Problema del servidor SMTP, no del colector | En Grafana → Alerting → Contact points → `email-cpd` → **Test**. Si falla, avisa a IT sobre la cuenta de correo configurada en `GRAFANA_SMTP_USER` (dentro de `.env`). |
| Nada de lo anterior funciona | — | Copia el mensaje de error exacto (de `journalctl` o de la pantalla) y pásalo a soporte técnico — con el error literal se resuelve mucho más rápido que describiéndolo de memoria. |

---

## 10. Dónde está todo (referencia rápida)

| Qué | Dónde |
|---|---|
| Raspberry Pi (IP) | `IP_DE_TU_RASPBERRY`, usuario SSH `cpd` |
| Dashboard de Grafana | `http://IP_DE_TU_RASPBERRY:3000` |
| InfluxDB (rara vez hace falta entrar directamente) | `http://IP_DE_TU_RASPBERRY:8086` |
| Carpeta de trabajo del proyecto (en la Pi) | `~/cpd-monitor` |
| Copia real que usa el servicio | `/opt/cpd-monitor` |
| Todas las contraseñas y tokens | fichero `.env` dentro de `~/cpd-monitor` (nunca se sube a GitHub) |
| Configuración de los sensores | `config.yaml` (tampoco se sube a GitHub) |
| Repositorio en GitHub | `github.com/alexio9910/cpd-monitor` |

> ⚠️ `.env` y `config.yaml` contienen contraseñas y tokens reales.
> Nunca los compartas por email/chat ni los subas a ningún sitio público.

---

## 11. Configurar una IP fija para la Raspberry Pi

Por defecto, la Raspberry Pi recibe una IP asignada automáticamente por tu
router (DHCP), que **puede cambiar** con el tiempo (por ejemplo, tras un
corte de luz largo o un reinicio del router). Si cambia, dejan de
funcionar: el enlace al dashboard que llevan los emails de alerta, el
acceso por SSH que tengas guardado, y `deploy.sh`. Conviene fijarla en
cuanto el sistema pase a producción de verdad.

Dos formas de hacerlo — elige la que puedas usar:

### Opción A — Reserva de IP en el router (recomendada)

La Pi sigue "hablando" DHCP con normalidad (cero riesgo de quedarse sin
red por una mala configuración local); simplemente le dices al router
"a este dispositivo dale siempre la misma IP".

1. Averigua la dirección MAC de la Raspberry Pi:
```bash
   ip link show eth0   # para conexión por cable
   ip link show wlan0  # para wifi
```
   Busca la línea `link/ether AA:BB:CC:DD:EE:FF` — esa es la MAC.
2. Entra en la web de administración de tu router (normalmente algo como
   `192.168.1.1` o `192.168.0.1` en el navegador).
3. Busca una sección "DHCP", "Reserva de IP" o "Static Leases" (el
   nombre exacto varía según la marca del router).
4. Añade una reserva: la MAC de la Pi → la IP que quieras que tenga
   siempre (lo más cómodo: la misma que ya tiene ahora, para no tener
   que cambiar nada más después).
5. Guarda y reinicia la Raspberry Pi.

> Si la red la gestiona el departamento de IT de la empresa y no tienes
> acceso al router, pídeles directamente esta reserva — es la opción más
> limpia y no requiere tocar nada en la Pi.

### Opción B — IP fija configurada en la propia Raspberry Pi

Si no puedes tocar el router. Las versiones actuales de Raspberry Pi OS
gestionan la red con **NetworkManager** — la guía antigua de editar
`/etc/dhcpcd.conf` ya no funciona en instalaciones recientes.

```bash
ssh cpd@IP_DE_TU_RASPBERRY
nmcli con show
```
Anota el nombre exacto de tu conexión (algo como `Wired connection 1`
para cable, o el nombre de tu red wifi).

```bash
sudo nmcli con mod "Wired connection 1" \
  ipv4.addresses IP_QUE_QUIERAS/24 \
  ipv4.gateway IP_DE_TU_ROUTER \
  ipv4.dns IP_DE_TU_ROUTER \
  ipv4.method manual

sudo nmcli con up "Wired connection 1"
```
Sustituye `"Wired connection 1"` por el nombre real del paso anterior,
`IP_QUE_QUIERAS` por la IP fija deseada (ej. `192.168.1.50`), y
`IP_DE_TU_ROUTER` por la IP de tu router (normalmente termina en `.1`).

Comprueba que se aplicó:
```bash
ip a show eth0
```

### Después de fijar la IP

Si la IP nueva es distinta a la que tenías, actualiza:

- `GRAFANA_PUBLIC_URL` en `.env` (el enlace de los emails de alerta):
```bash
  nano .env   # cambia GRAFANA_PUBLIC_URL=http://IP_ANTIGUA:3000
  docker compose up -d --force-recreate grafana
```
- `PI_HOST` en el `.env` de tu WSL, si usas `deploy.sh`.
- Cualquier acceso directo o gestor de contraseñas apuntando a la IP
  antigua.

---

## 12. Glosario

| Término | Qué es |
|---|---|
| Colector | El programa que lee los sensores cada minuto (corre como servicio en la Pi). |
| InfluxDB | La base de datos donde se guarda el histórico de lecturas. |
| Grafana | La web que dibuja las gráficas y manda las alertas. |
| Contact point | En Grafana, a quién/cómo se envía una alerta (en nuestro caso, email). |
| SMTP | El servidor de correo que Grafana usa para poder enviar emails. |
| BLE / Bluetooth Low Energy | El tipo de Bluetooth de bajo consumo que usan los sensores Sensirion. |
| Systemd / servicio | El mecanismo de Linux que arranca el colector solo y lo reinicia si falla. |
