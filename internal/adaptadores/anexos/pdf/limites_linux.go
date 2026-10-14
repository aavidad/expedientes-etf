package pdf

import "syscall"

// limitarProceso nunca sube el máximo que ya impone el sistema: un usuario sin
// privilegios no puede elevarlo y el lector fallaría al arrancar.
func limitarProceso() error {
	// Go reserva arenas virtuales mayores que su memoria residente. DATA
	// también cuenta los mmap de lectura/escritura, incluidos los del runtime.
	for recurso, limite := range map[int]uint64{
		syscall.RLIMIT_AS: 4 << 30, syscall.RLIMIT_CPU: 3,
		syscall.RLIMIT_DATA: 256 << 20,
		syscall.RLIMIT_CORE: 0, syscall.RLIMIT_FSIZE: 0,
	} {
		var actual syscall.Rlimit
		if err := syscall.Getrlimit(recurso, &actual); err != nil {
			return err
		}
		limite = min(limite, actual.Cur, actual.Max)
		if err := syscall.Setrlimit(recurso, &syscall.Rlimit{Cur: limite, Max: limite}); err != nil {
			return err
		}
	}
	return nil
}
