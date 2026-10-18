// Programa principal del Hospital de los Costeños con Narcolepsia.
// Solo arma el escenario e imprime; la lógica vive en el paquete hospital.
//
//	go run .          escenario de demostración
//	go run . -sim     simulación concurrente
//	go run . -web     interfaz web en http://localhost:8080
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"sort"
	"time"

	"costenos-narcolepsia/hospital"
	"costenos-narcolepsia/simulation"
	"costenos-narcolepsia/web"
)

func main() {
	simMode := flag.Bool("sim", false, "corre la simulación concurrente")
	webMode := flag.Bool("web", false, "levanta la interfaz web")
	port := flag.Int("port", 8080, "puerto de la interfaz web")
	simFor := flag.Duration("dur", 4*time.Second, "duración de la simulación")
	flag.Parse()

	h, err := newDemoHospital()
	if err != nil {
		log.Fatalf("no se pudo armar el hospital: %v", err)
	}

	switch {
	case *webMode:
		addr := fmt.Sprintf(":%d", *port)
		fmt.Printf("🌴 %s atendiendo en http://localhost%s (Ctrl+C para cerrar)\n", h.Name(), addr)
		log.Fatal(web.ListenAndServe(addr, h))
	case *simMode:
		runSimulation(h, *simFor)
	default:
		runScenario(h)
	}
}

// newDemoHospital arma el hospital de ejemplo sin imprimir nada.
// Lo usan los tres modos: escenario, simulación y web.
func newDemoHospital() (*hospital.Hospital, error) {
	h := hospital.NewHospital("Hospital de los Costeños con Narcolepsia")

	doctors := []*hospital.Doctor{
		hospital.NewDoctor("D-001", "Rafaela Vides", 45, "Neurología del sueño"),
		hospital.NewDoctor("D-002", "Aníbal Mercado", 52, "Medicina interna"),
	}
	for _, d := range doctors {
		if err := h.HireDoctor(d); err != nil {
			return nil, err
		}
	}
	if err := h.HireStaff(hospital.NewOrderly("O-001", "Toño Rangel", 38)); err != nil {
		return nil, err
	}

	for _, n := range []int{101, 102, 103} {
		r, err := hospital.NewRoom(n, 1)
		if err != nil {
			return nil, err
		}
		if err := h.AddRoom(r); err != nil {
			return nil, err
		}
	}

	patients := []*hospital.Patient{
		hospital.NewPatient("P-001", "Wilfrido Berrio", 54, hospital.Severe),
		hospital.NewPatient("P-002", "Yeimy Padilla", 31, hospital.Moderate),
		hospital.NewPatient("P-003", "Karen Ospina", 27, hospital.Mild),
		hospital.NewPatient("P-004", "Eliécer Mendoza", 68, hospital.Severe),
		hospital.NewPatient("P-005", "Yuranis Barrios", 40, hospital.Moderate),
	}
	for _, p := range patients {
		if err := h.AdmitPatient(p); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// runScenario corre el escenario obligatorio de la entrega, paso a paso.
func runScenario(h *hospital.Hospital) {
	fmt.Printf("🏥 %s abre sus puertas, ¡ajá!\n", h.Name())
	fmt.Println("   2 doctores, 1 camillero, 3 salas de 1 cama y 5 pacientes con un sueño berraco.")

	fmt.Println("\n== Ataques de sueño ==")
	attacks := []struct{ id, location string }{
		{"P-001", "la cafetería, con la cara metida en la arepa de huevo"},
		{"P-002", "la fila de radiología"},
		{"P-003", "el pasillo del segundo piso"},
		{"P-004", "la puerta de urgencias, bloqueando la entrada"},
	}
	for _, a := range attacks {
		rec, err := h.HandleSleepAttack(a.id, a.location)
		switch {
		case errors.Is(err, hospital.ErrNoRoomAvailable):
			fmt.Println("  ", rec.Summary())
			fmt.Printf("   ⚠️  ¡Ay hombe! %v. Que nadie lo pise.\n", err)
		case err != nil:
			fmt.Printf("   ❌ error inesperado: %v\n", err)
		default:
			fmt.Println("  ", rec.Summary())
		}
	}

	fmt.Println("\n== Patients asleep in hallway (justo después del fallo) ==")
	printHallway(h.PatientsInHallway())

	fmt.Println("\n== Yeimy se despierta ==")
	if err := h.WakePatient("P-002"); err != nil {
		fmt.Printf("   ❌ %v\n", err)
	} else {
		fmt.Println("   Yeimy Padilla (P-002) se despertó preguntando \"¿y yo qué hago aquí?\" y liberó su sala.")
	}
	p4, err := h.FindPatient("P-004")
	if err != nil {
		fmt.Printf("   ❌ %v\n", err)
		return
	}
	room, assignErr := h.AssignRoom(p4)

	fmt.Println("\n=================== RESULTADOS ===================")
	fmt.Println("\n== Patients asleep in hallway ==")
	printHallway(h.PatientsInHallway())

	for _, d := range h.Doctors() {
		fmt.Printf("\n== Episodes attended by Dr. %s ==\n", d.Name())
		eps := d.MyEpisodes()
		if len(eps) == 0 {
			fmt.Println("   (ninguno, se la pasó tomando tinto)")
		}
		for _, e := range eps {
			fmt.Println("  ", e.Summary())
		}
	}

	fmt.Printf("\n== Assigning room to %s ==\n", p4.ID())
	if assignErr != nil {
		fmt.Printf("   ⚠️  %v\n", assignErr)
	} else {
		fmt.Printf("   %s ahora está %s en la sala %d (episodio de seguimiento registrado).\n",
			p4.Name(), p4.State(), room.Number())
	}

	fmt.Println("\n== Severity report (Severe patients) ==")
	report := h.SevereReport()
	severe := make([]*hospital.Patient, 0, len(report))
	for p := range report {
		severe = append(severe, p)
	}
	sort.Slice(severe, func(i, j int) bool { return severe[i].ID() < severe[j].ID() })
	for _, p := range severe {
		fmt.Printf("   %s %-16s episodios hoy: %d\n", p.ID(), p.Name(), report[p])
	}
}

func printHallway(patients []*hospital.Patient) {
	if len(patients) == 0 {
		fmt.Println("   Pasillo despejado, se puede caminar tranquilo.")
		return
	}
	for _, p := range patients {
		fmt.Printf("   %s %s (%s), roncando en %s\n", p.ID(), p.Name(), p.Level(), p.Location())
	}
}

// runSimulation corre la simulación concurrente e imprime el resumen.
func runSimulation(h *hospital.Hospital, d time.Duration) {
	fmt.Printf("🌀 Simulando %v de caos en %s...\n", d, h.Name())
	stats := simulation.Run(h, d)
	snap := h.Snapshot()

	occupied := 0
	for _, r := range snap.Rooms {
		if r.State == hospital.Occupied.String() {
			occupied++
		}
	}
	fmt.Println("\n== Resumen de la simulación ==")
	fmt.Printf("   Ataques de sueño:           %d\n", stats.Attacks)
	fmt.Printf("   Sin sala en el ataque:      %d\n", stats.NoRoom)
	fmt.Printf("   Despertares:                %d\n", stats.Wakes)
	fmt.Printf("   Reubicaciones del pasillo:  %d\n", stats.Relocations)
	fmt.Printf("   Episodios totales:          %d\n", snap.TotalEpisodes)
	fmt.Printf("   Salas ocupadas al final:    %d de %d\n", occupied, len(snap.Rooms))
	fmt.Printf("   Pacientes en el pasillo:    %d %v\n", len(snap.Hallway), snap.Hallway)
	for _, s := range snap.Staff {
		fmt.Printf("   %-10s %-16s atendió %d episodios\n", s.Role, s.Name, s.Episodes)
	}
}
