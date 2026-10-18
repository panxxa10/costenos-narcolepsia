# Hospital de los Costeños con Narcolepsia 🌴😴

Trabajo de Programación Orientada a Objetos, Universidad EIA.
Sistema en Go (solo librería estándar) que lleva el control de pacientes costeños con narcolepsia: dónde se durmieron, quién los atendió y a qué sala fueron a parar antes de que bloqueen un pasillo.

## Integrantes

- [MI NOMBRE]

## Requisitos

- Go 1.21 o superior (se desarrolló y probó con Go 1.27.0, darwin/arm64).
- Sin dependencias externas.

## Cómo correr

```bash
go run .                 # escenario de demostración obligatorio
go run -race . -sim      # simulación concurrente (~4 s) con detector de carreras
go run . -sim -dur 10s   # simulación más larga
go run . -web            # interfaz web en http://localhost:8080
go run . -web -port 9000 # otro puerto
```

## Cómo probar

```bash
go test ./...
go test -race ./...
gofmt -l .        # no debe imprimir nada
go vet ./...      # sin advertencias
```

El detector de carreras sí estaba disponible en el entorno de desarrollo (cgo activo, clang instalado). `go test -race ./...` y `go run -race . -sim` se ejecutaron y pasaron sin reportar carreras.

## Estructura

```
costenos-narcolepsia/
  go.mod
  main.go              escenario + flags -sim / -web; solo arma e imprime
  hospital/            modelo y lógica, sin nada de presentación
    hospital.go        Hospital, errores centinela, interfaz Attender, consultas, Snapshot
    patient.go         Person, Patient, NarcolepsyLevel, PatientState
    doctor.go          Doctor (implementa Attender)
    orderly.go         Orderly / camillero (implementa Attender)
    room.go            Room, RoomState
    episode.go         EpisodeRecord
    hospital_test.go   tests (incluye el concurrente)
  simulation/          bonus: una goroutine por paciente
  web/                 bonus: API REST + frontend embebido
  README.md, AI_USAGE.md, DEFENSA.md
```

## API web

| Método | Ruta | Cuerpo | Respuesta |
|---|---|---|---|
| GET | `/api/state` | — | `Snapshot` del hospital en JSON |
| POST | `/api/attack` | `{"patientId":"P-001","location":"la cafetería"}` | `{"ok":true,"message":...,"warning":...}` |
| POST | `/api/wake` | `{"patientId":"P-001"}` | `{"ok":true,"message":...,"episodes":[...]}` |
| GET | `/` | — | frontend (`web/static/index.html` embebido) |

Errores: 400 (JSON inválido o campos faltantes), 404 (paciente no existe), 405 (método equivocado), 409 (ya dormido / ya despierto), 503 (sin personal). Siempre con JSON `{"ok":false,"message":...}`.

## Decisiones de diseño

- **Campos en minúscula.** Todo el estado (`state`, `room`, `occupants`, `history`...) es no exportado. Desde otro paquete solo se puede leer con métodos (`State()`, `Room()`) y cambiar con métodos que mantienen las reglas: una sala no puede quedar "disponible" con gente adentro, un paciente no puede tener cama sin estar en la lista de ocupantes, y todo cambio pasa por el mutex del Hospital.
- **Person embebido por valor** en `Patient`, `Doctor` y `Orderly`. Cada uno tiene su propia copia de id/nombre/edad (nadie la comparte), el valor cero sirve, no hay riesgo de puntero nil, y los métodos `ID()`, `Name()`, `Age()` se promueven solos. Gracias a eso `Doctor` y `Orderly` cumplen la parte `ID()/Name()` de `Attender` sin escribir nada.
- **Receptores puntero** en `Hospital`, `Patient`, `Doctor`, `Orderly` y `Room`, porque sus métodos modifican el struct (con receptor valor se modificaría una copia) y porque `Hospital` tiene un `sync.Mutex`, que no se debe copiar. `Person` y `EpisodeRecord` usan receptor valor: son pequeños y no se modifican después de creados.
- **Un mutex en el Hospital** (y no uno por sala o paciente). Es lo más sencillo que garantiza que "buscar sala libre + ocuparla" sea atómico. Los métodos públicos toman el lock con `defer h.mu.Unlock()` y delegan en helpers `...Locked` que nunca lo toman, porque `sync.Mutex` no es reentrante. `Patient`, `Doctor` y `Room` no son seguros por sí solos: desde varias goroutines se usan solo a través del Hospital.
- **map en `SevereReport`.** Permite contar en una sola pasada por el historial (`report[e.patient]++`) con búsqueda O(1) y sin claves repetidas. Como el orden de un map es aleatorio, `main` ordena las claves por ID antes de imprimir. Los pacientes severos sin episodios aparecen con 0.
- **Split de paquetes.** `hospital` no importa a nadie del proyecto; `simulation` y `web` importan a `hospital`; `main` arma todo. Así se agregaron la simulación y la web sin modificar `hospital` (en el historial de git, `simulation/` y `web/` llegan después de las consultas y no tocan el paquete hospital).
- **Episodio de seguimiento al reubicar.** Cuando un paciente del pasillo consigue sala después, queda un nuevo `EpisodeRecord` con la sala. Así el historial dice la verdad: en el momento del ataque no hubo sala (episodio sin sala), y luego sí (episodio de seguimiento). Por eso P-004 tiene 2 episodios en el reporte.

### Decisiones que tomé donde el enunciado dejaba dudas

1. **Orden dentro de `HandleSleepAttack`: la sala se asigna antes de llamar a `Attend`.** El enunciado lista "Attend, luego AssignRoom". Pero `Attend` es quien crea el `EpisodeRecord` y el doctor lo guarda en su propia lista; si la sala se asignara después, la copia del doctor quedaría sin sala aunque el paciente sí la tuviera. Asignando primero, el Attender ve la sala del paciente y el registro del doctor y el del historial coinciden. El turno sí se despacha antes de intentar la sala.
2. **`AssignRoom` público registra el episodio de seguimiento**, porque se usa para reubicar a pacientes del pasillo. Dentro de `HandleSleepAttack` se usa el helper interno `assignRoomLocked`, que no crea episodio extra.
3. **`WakePatient` solo despierta y libera la sala.** La reubicación del pasillo es explícita: en `main` con `AssignRoom(P-004)` (para usar la consulta 5.3) y en la web/simulación con `RelocateHallwayPatients()`.
4. **Paso 4 y bloque final.** El `AssignRoom(P-004)` se ejecuta en el paso 4, antes del bloque de resultados; el encabezado `== Assigning room to P-004 ==` imprime el resultado de esa llamada. Así el pasillo final sale vacío y `MyEpisodes` ya incluye el seguimiento.
5. **ID de episodios** con un contador atómico del paquete (`EP-001`, `EP-002`...), porque `Attend` crea el registro y no recibe el hospital.
6. **`Person` vive en `patient.go`** y `Attender` y los errores en `hospital.go`, para respetar la estructura de archivos pedida.
7. **`HireDoctor` y `HireStaff(a Attender)`**: doctores van a la lista de doctores y a la rotación; camilleros (o cualquier otro Attender) solo a la rotación.
8. **`DiagnosePatient` falla si el paciente ya tiene doctor.** Si otro doctor atiende un seguimiento, atiende pero no le cambia el doctor asignado.
9. **Número de sala 0 = sin sala** en `EpisodeRecord` (por eso `NewRoom` exige número positivo).
10. **El `Snapshot` no tiene etiquetas `json`** (eso sería presentación dentro de `hospital`), así que el JSON usa los nombres de campo de Go (`ID`, `Name`, `State`...).
11. **En la web, un ataque sin sala responde 200** con `warning`, porque la petición sí se cumplió: el episodio quedó registrado.
12. **`go.mod` dice `go 1.21`** (mínimo pedido). Por eso el router usa `http.ServeMux` clásico y verifica el método a mano (los patrones `"GET /ruta"` son de Go 1.22), y en las goroutines la variable del `for` se pasa como parámetro (antes de Go 1.22 la variable del ciclo se comparte).
13. **Al despertar, el paciente vuelve a "consulta externa"**, para no quedar con la ubicación de una sala que ya es de otro.
