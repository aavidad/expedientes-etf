package main

import "testing"

func TestLeerProxies(t *testing.T) {
	redes, err := leerProxies(" 10.250.4.0/28, 192.0.2.1/32 ,")
	if err != nil || len(redes) != 2 || redes[0].String() != "10.250.4.0/28" {
		t.Fatalf("redes válidas: %v %v", redes, err)
	}
	if redes, err := leerProxies(""); err != nil || len(redes) != 0 {
		t.Fatalf("vacía: %v %v", redes, err)
	}
	for _, malo := range []string{"10.0.0.1", "10.0.0.1/24", "proxy", "10.0.0.0/33"} {
		if _, err := leerProxies(malo); err == nil {
			t.Fatalf("%q debe rechazarse", malo)
		}
	}
}
