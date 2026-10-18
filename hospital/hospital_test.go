package hospital

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

// newTestHospital arma un hospital con 1 doctor, 1 camillero, las salas
// pedidas (capacidad 1, numeradas desde 101) y los pacientes dados.
func newTestHospital(t *testing.T, rooms int, patients ...*Patient) (*Hospital, *Doctor, *Orderly) {
	t.Helper()
	h := NewHospital("Hospital de prueba")
	d := NewDoctor("D-1", "Rafaela Vides", 45, "Neurología")
	o := NewOrderly("O-1", "Toño Rangel", 38)
	if err := h.HireDoctor(d); err != nil {
		t.Fatal(err)
	}
	if err := h.HireStaff(o); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < rooms; i++ {
		r, err := NewRoom(101+i, 1)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.AddRoom(r); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range patients {
		if err := h.AdmitPatient(p); err != nil {
			t.Fatal(err)
		}
	}
	return h, d, o
}

func TestAssignRoomWhenOneIsFree(t *testing.T) {
	p := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	h, _, _ := newTestHospital(t, 1, p)
	p.SufferSleepAttack("la cafetería")

	room, err := h.AssignRoom(p)
	if err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if room.Number() != 101 || room.State() != Occupied {
		t.Errorf("sala = %d (%s), quería 101 ocupada", room.Number(), room.State())
	}
	if p.State() != AsleepInBed || p.Room() != room {
		t.Errorf("paciente quedó %s en %v, quería dormido en cama en la 101", p.State(), p.Room())
	}
}

func TestAssignRoomFailsWhenNoRoomIsFree(t *testing.T) {
	p1 := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	p2 := NewPatient("P-2", "Eliécer Mendoza", 68, Severe)
	h, _, _ := newTestHospital(t, 1, p1, p2)
	if _, err := h.HandleSleepAttack("P-1", "la cafetería"); err != nil {
		t.Fatal(err)
	}
	p2.SufferSleepAttack("urgencias")

	room, err := h.AssignRoom(p2)
	if !errors.Is(err, ErrNoRoomAvailable) {
		t.Fatalf("err = %v, quería ErrNoRoomAvailable", err)
	}
	if room != nil {
		t.Errorf("no debía inventarse una sala, llegó la %d", room.Number())
	}
	if p2.State() != AsleepInHallway {
		t.Errorf("estado = %s, quería %s", p2.State(), AsleepInHallway)
	}
}

func TestHandleSleepAttackWithoutRoomStillRegistersEpisode(t *testing.T) {
	p := NewPatient("P-1", "Karen Ospina", 27, Mild)
	h, _, _ := newTestHospital(t, 0, p)

	rec, err := h.HandleSleepAttack("P-1", "radiología")
	if !errors.Is(err, ErrNoRoomAvailable) {
		t.Fatalf("err = %v, quería ErrNoRoomAvailable", err)
	}
	if rec.HasRoom() {
		t.Error("el episodio no debía tener sala")
	}
	if got := len(h.History()); got != 1 {
		t.Errorf("historial con %d episodios, quería 1", got)
	}
	if p.State() != AsleepInHallway || p.Location() != "radiología" {
		t.Errorf("paciente %s en %q, quería dormido en el pasillo de radiología", p.State(), p.Location())
	}
}

func TestPatientsInHallway(t *testing.T) {
	p1 := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	p2 := NewPatient("P-2", "Yeimy Padilla", 31, Moderate)
	p3 := NewPatient("P-3", "Karen Ospina", 27, Mild)
	h, _, _ := newTestHospital(t, 1, p1, p2, p3)
	h.HandleSleepAttack("P-1", "cafetería")        // le toca la única sala
	h.HandleSleepAttack("P-2", "fila de farmacia") // pasillo

	hallway := h.PatientsInHallway()
	if len(hallway) != 1 || hallway[0] != p2 {
		t.Fatalf("pasillo = %v, quería solo a P-2", hallway)
	}
}

func TestSevereReport(t *testing.T) {
	p1 := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	p2 := NewPatient("P-2", "Yeimy Padilla", 31, Moderate)
	p3 := NewPatient("P-3", "Eliécer Mendoza", 68, Severe)
	h, _, _ := newTestHospital(t, 3, p1, p2, p3)
	h.HandleSleepAttack("P-1", "cafetería")
	h.WakePatient("P-1")
	h.HandleSleepAttack("P-1", "parqueadero")
	h.HandleSleepAttack("P-2", "ascensor")

	report := h.SevereReport()
	if len(report) != 2 {
		t.Fatalf("el reporte tiene %d pacientes, quería 2 (solo severos)", len(report))
	}
	if report[p1] != 2 {
		t.Errorf("P-1 tiene %d episodios, quería 2", report[p1])
	}
	if n, ok := report[p3]; !ok || n != 0 {
		t.Errorf("P-3 debía aparecer con 0 episodios, llegó %d (está=%v)", n, ok)
	}
	if _, ok := report[p2]; ok {
		t.Error("P-2 es moderado y no debía aparecer")
	}
}

func TestPersonIsEmbedded(t *testing.T) {
	p := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	d := NewDoctor("D-1", "Rafaela Vides", 45, "Neurología")
	o := NewOrderly("O-1", "Toño Rangel", 38)

	if p.ID() != "P-1" || p.Name() != "Wilfrido Berrio" || p.Age() != 54 {
		t.Errorf("métodos promovidos de Person en Patient: %s %s %d", p.ID(), p.Name(), p.Age())
	}
	if d.Person.Name() != d.Name() || o.Person.ID() != o.ID() {
		t.Error("Doctor y Orderly deben exponer los métodos de su Person embebido")
	}
}

func TestDispatchIsRoundRobinAndPolymorphic(t *testing.T) {
	h, d, o := newTestHospital(t, 0)
	want := []Attender{d, o, d}
	for i, w := range want {
		got, err := h.DispatchAttender()
		if err != nil {
			t.Fatal(err)
		}
		if got != w {
			t.Errorf("turno %d: salió %s, quería %s", i, got.Name(), w.Name())
		}
	}

	// Los dos tipos se usan por la interfaz, sin saber cuál es cuál.
	p := NewPatient("P-1", "Karen Ospina", 27, Mild)
	p.SufferSleepAttack("pasillo")
	for _, a := range []Attender{d, o} {
		rec, err := a.Attend(p, "pasillo")
		if err != nil {
			t.Fatalf("%s: %v", a.Name(), err)
		}
		if rec.Attending() != a {
			t.Errorf("el episodio dice que lo atendió %s, quería %s", rec.Attending().Name(), a.Name())
		}
	}
}

func TestDispatchWithoutStaff(t *testing.T) {
	h := NewHospital("vacío")
	if _, err := h.DispatchAttender(); !errors.Is(err, ErrNoStaff) {
		t.Errorf("err = %v, quería ErrNoStaff", err)
	}
}

func TestDoctorMyEpisodes(t *testing.T) {
	p1 := NewPatient("P-1", "Wilfrido Berrio", 54, Severe)
	p2 := NewPatient("P-2", "Yeimy Padilla", 31, Moderate)
	p3 := NewPatient("P-3", "Karen Ospina", 27, Mild)
	h, d, o := newTestHospital(t, 3, p1, p2, p3)
	h.HandleSleepAttack("P-1", "cafetería") // doctor
	h.HandleSleepAttack("P-2", "ascensor")  // camillero
	h.HandleSleepAttack("P-3", "farmacia")  // doctor otra vez

	eps := d.MyEpisodes()
	if len(eps) != 2 || eps[0].Patient() != p1 || eps[1].Patient() != p3 {
		t.Fatalf("MyEpisodes = %d episodios, quería los de P-1 y P-3", len(eps))
	}
	if p1.AssignedDoctor() != d || p2.AssignedDoctor() != nil {
		t.Error("solo el doctor diagnostica; el camillero no asigna doctor")
	}
	if o.Transfers() != 1 {
		t.Errorf("el camillero hizo %d traslados, quería 1", o.Transfers())
	}
}

func TestWakePatientReleasesRoomAndHallwayIsRelocated(t *testing.T) {
	p1 := NewPatient("P-1", "Yeimy Padilla", 31, Moderate)
	p2 := NewPatient("P-2", "Eliécer Mendoza", 68, Severe)
	h, _, _ := newTestHospital(t, 1, p1, p2)
	h.HandleSleepAttack("P-1", "radiología")
	h.HandleSleepAttack("P-2", "urgencias")

	if err := h.WakePatient("P-1"); err != nil {
		t.Fatal(err)
	}
	if p1.State() != Awake || p1.Room() != nil || p1.Location() != "consulta externa" {
		t.Errorf("P-1 quedó %s con sala %v", p1.State(), p1.Room())
	}
	moved := h.RelocateHallwayPatients()
	if len(moved) != 1 || moved[0].RoomNumber() != 101 || p2.State() != AsleepInBed {
		t.Fatalf("P-2 debía pasar a la 101 con episodio de seguimiento; moved=%d estado=%s", len(moved), p2.State())
	}
	if got := h.SevereReport()[p2]; got != 2 {
		t.Errorf("P-2 tiene %d episodios, quería 2 (ataque + seguimiento)", got)
	}
	if err := h.WakePatient("P-1"); !errors.Is(err, ErrAlreadyAwake) {
		t.Errorf("err = %v, quería ErrAlreadyAwake", err)
	}
}

func TestAlreadyAsleepAndUnknownPatient(t *testing.T) {
	p := NewPatient("P-1", "Karen Ospina", 27, Mild)
	h, _, _ := newTestHospital(t, 1, p)
	h.HandleSleepAttack("P-1", "cafetería")
	if _, err := h.HandleSleepAttack("P-1", "cafetería"); !errors.Is(err, ErrAlreadyAsleep) {
		t.Errorf("err = %v, quería ErrAlreadyAsleep", err)
	}
	if _, err := h.HandleSleepAttack("P-99", "cafetería"); !errors.Is(err, ErrPatientNotFound) {
		t.Errorf("err = %v, quería ErrPatientNotFound", err)
	}
}

func TestRoomOccupyAndRelease(t *testing.T) {
	if _, err := NewRoom(101, 0); !errors.Is(err, ErrInvalidRoom) {
		t.Errorf("capacidad 0: err = %v, quería ErrInvalidRoom", err)
	}
	r, _ := NewRoom(101, 1)
	p1 := NewPatient("P-1", "A", 20, Mild)
	p2 := NewPatient("P-2", "B", 20, Mild)
	if err := r.Occupy(p1); err != nil {
		t.Fatal(err)
	}
	if err := r.Occupy(p2); !errors.Is(err, ErrRoomFull) {
		t.Errorf("err = %v, quería ErrRoomFull", err)
	}
	if err := r.Release(p2); !errors.Is(err, ErrNotInRoom) {
		t.Errorf("err = %v, quería ErrNotInRoom", err)
	}
	if err := r.Release(p1); err != nil || !r.IsAvailable() {
		t.Errorf("después de liberar: err=%v disponible=%v", err, r.IsAvailable())
	}
}

// TestConcurrentAssignRoom: 20 goroutines piden sala a la vez y solo hay 3.
// Correr con go test -race ./...
func TestConcurrentAssignRoom(t *testing.T) {
	const n = 20
	var patients []*Patient
	for i := 0; i < n; i++ {
		patients = append(patients, NewPatient(fmt.Sprintf("P-%02d", i), "Paciente", 30, Moderate))
	}
	h, _, _ := newTestHospital(t, 3, patients...)
	for _, p := range patients {
		p.SufferSleepAttack("pasillo") // antes de lanzar goroutines, sin concurrencia
	}

	type result struct {
		room *Room
		err  error
	}
	results := make(chan result, n)
	var wg sync.WaitGroup
	for _, p := range patients {
		wg.Add(1)
		go func(p *Patient) {
			defer wg.Done()
			room, err := h.AssignRoom(p)
			results <- result{room, err}
		}(p)
	}
	wg.Wait()
	close(results)

	ok, noRoom := 0, 0
	seen := make(map[int]bool)
	for r := range results {
		switch {
		case r.err == nil:
			ok++
			if seen[r.room.Number()] {
				t.Errorf("la sala %d se asignó dos veces", r.room.Number())
			}
			seen[r.room.Number()] = true
		case errors.Is(r.err, ErrNoRoomAvailable):
			noRoom++
		default:
			t.Errorf("error inesperado: %v", r.err)
		}
	}
	if ok != 3 || noRoom != n-3 {
		t.Errorf("éxitos=%d sin sala=%d, quería 3 y %d", ok, noRoom, n-3)
	}
}
