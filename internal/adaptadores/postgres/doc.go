// Package postgres proporciona el pool común y las transacciones con contexto
// RLS para los adaptadores de los módulos. DesdeContexto es el único acceso a
// consultas de repositorio; se usa con el contexto recibido de Ejecutar.
//
// Configuración: ETF_BD_HOST (127.0.0.1), ETF_BD_PUERTO (5432), ETF_BD_NOMBRE
// (etf), ETF_BD_USUARIO (sólo etf_app), ETF_BD_CONTRASENA (obligatoria),
// ETF_BD_MAX_CONEXIONES (4, entre 1 y 64), ETF_BD_SSLMODE (disable para la red
// interna de Compose, o verify-full) y ETF_BD_SSLROOTCERT (CA para TLS).
// No se leen credenciales de migración ni opciones de sesión del entorno PG*.
// cmd/etf retira PG* una vez antes de crear el pool y las goroutines, ya que
// el parser de pgx consulta PGSERVICE/PGSERVICEFILE. Otros puntos de composición
// deben cumplir la misma precondición al usar Nueva.
//
// Pruebas reales: ETF_POSTGRES_INTEGRACION=1 go test -v ./internal/adaptadores/postgres
// con ETF_BD_* del PostgreSQL 18 desechable de compose.yaml. Al omitir esa
// variable sólo se omiten las pruebas de integración. No se crean tablas.
package postgres
