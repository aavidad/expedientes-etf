import { useEffect, useRef, useState, type ReactNode } from 'react';
import { useButton } from 'react-aria';
import { pantallas } from './pantallas';
import { useEquipoPersona, seccionCoordinacion } from './ui3/navegacion';
import { menuPermitido } from './registro-pantallas';

export type Acceso = {
  estado: 'sin_sesion' | 'pendiente_alta' | 'activo';
  csrf: string;
  usuario?: { id: string; cuenta: string; nombre: string };
  perfiles?: string[];
  equipo_id?: string;
  pantallas?: { ruta: string; perfiles: string[]; clave: string }[];
};
type Textos = (clave: string) => string;
const perfiles = ['coordinacion', 'profesional_etf', 'consulta', 'administracion'];
const paginaActual = 'page' as const;

export function Armazon({ t, acceso, preferencias, salir, ocupado, ruta, navegar, children }: {
  t: Textos; acceso?: Acceso; preferencias?: ReactNode; salir?: () => void;
  ocupado?: boolean; ruta?: string; navegar?: (ruta: string) => void; children: ReactNode;
}) {
  const [compacto, ponerCompacto] = useState(true);
  const [abierto, ponerAbierto] = useState(false);
  const botonMenu = useRef<HTMLButtonElement>(null);
  const botonSalir = useRef<HTMLButtonElement>(null);
  const persona = useRef<HTMLDetailsElement>(null);
  const resumenPersona = useRef<HTMLElement>(null);
  const { buttonProps: propsMenu } = useButton({ onPress: () => ponerAbierto((v) => !v) }, botonMenu);
  const { buttonProps: propsSalir } = useButton({ onPress: salir, isDisabled: ocupado }, botonSalir);
  const disponibles = perfiles.filter((p) => acceso?.estado === 'activo' && acceso.perfiles?.includes(p));
  const rutasPaquetes = ['/admin/paquetes', '/admin/idiomas', '/admin/temas'];
  const permitidos = menuPermitido(pantallas, acceso);
  const entradaPaquetes = permitidos.find((p) => rutasPaquetes.includes(p.ruta));
  const enlaces = permitidos.filter((p) => !rutasPaquetes.includes(p.ruta) || p === entradaPaquetes);
  const conMenu = enlaces.length > 0;
  const conSesion = acceso && acceso.estado !== 'sin_sesion';
  const nombreMenu = t('usuarios.menu');
  const etiquetaPersona = compacto ? t('comun.persona_menu') : undefined;
  const nombre = acceso?.usuario?.nombre ?? t('identidad.pendiente_titulo');
  const roles = new Intl.ListFormat(document.documentElement.lang, { style: 'short', type: 'conjunction' }).format(disponibles.map((p) => t(`usuarios.perfil_${p}`)));
  const equipoPersona = useEquipoPersona(acceso, document.documentElement.lang);
  const equipo = acceso?.equipo_id ? equipoPersona.nombre || t(equipoPersona.error ? 'comun.equipo_error' : 'comun.cargando') : '';

  useEffect(() => {
    const observar = new ResizeObserver(() => {
      const umbral = getComputedStyle(document.documentElement).getPropertyValue('--ancho-menu');
      ponerCompacto(document.documentElement.clientWidth < parseFloat(umbral) * 4);
    });
    observar.observe(document.documentElement);
    return () => observar.disconnect();
  }, []);
  useEffect(() => { ponerAbierto(false); if (persona.current) persona.current.open = false; }, [ruta, compacto]);
  function cerrarMenu() { ponerAbierto(false); botonMenu.current?.focus(); }
  function cerrarPersona() { if (persona.current) persona.current.open = false; resumenPersona.current?.focus(); }

  return <div className="armazon">
    <a className="saltar-contenido" href="#contenido" onClick={() => document.getElementById('contenido')?.focus()}>{t('comun.saltar_contenido')}</a>
    <header className="cabecera-app" data-compacto={compacto}>
      <div className="marca-app"><span className="marca" aria-hidden="true">{t('comun.marca')}</span><span>{t('comun.titulo') || document.title}</span></div>
      <div className="acciones-cabecera">
        {compacto && conMenu && <button {...propsMenu} ref={botonMenu} className="boton-secundario" aria-expanded={abierto} aria-controls="menu-principal">{t('comun.menu')}</button>}
        {conSesion ? <details className="persona-conectada" ref={persona}
          onKeyDown={(evento) => { if (evento.key === 'Escape') { evento.preventDefault(); cerrarPersona(); } }}
          onBlur={(evento) => { if (!evento.currentTarget.contains(evento.relatedTarget) && persona.current) persona.current.open = false; }}>
          <summary ref={resumenPersona} aria-label={etiquetaPersona}>
            {compacto ? t('comun.persona_menu') : <span className="resumen-persona"><strong>{nombre}</strong><small>{roles}{equipo && <span className="equipo-persona">{equipo}</span>}</small></span>}
          </summary>
          <div className="panel-persona">
            <strong>{nombre}</strong>
            <ul className="perfiles-persona">{disponibles.map((p) => <li key={p}>{t(`usuarios.perfil_${p}`)}</li>)}</ul>
            {equipo && <small className="equipo-persona">{equipo}</small>}
            {equipoPersona.error && <button type="button" onClick={equipoPersona.recargar}>{t('comun.reintentar')}</button>}
            {preferencias}
            <button {...propsSalir} ref={botonSalir} className="boton-secundario">{t('identidad.salir')}</button>
          </div>
        </details> : preferencias}
      </div>
    </header>
    <div className="cuerpo-app" data-compacto={compacto}>
      {conMenu && <aside className="lateral-app" hidden={compacto && !abierto}>
        <nav id="menu-principal" aria-label={nombreMenu} onKeyDown={(evento) => { if (compacto && evento.key === 'Escape') { evento.preventDefault(); cerrarMenu(); } }}>
          {disponibles.flatMap((p) => p === 'coordinacion' ? ['derivaciones', 'expedientes', 'informes', 'configuracion'].map((seccion) => ({ perfil: p, seccion })) : [{ perfil: p, seccion: p }]).map(({ perfil: p, seccion }) => {
            const grupo = enlaces.filter((e) => e.perfiles.find((perfil) => disponibles.includes(perfil)) === p && (p !== 'coordinacion' || seccionCoordinacion(e.ruta) === seccion));
            return grupo.length > 0 && <section key={seccion} aria-labelledby={`menu-${seccion}`}>
              <h2 id={`menu-${seccion}`}>{t(p === 'coordinacion' ? `comun.menu.${seccion}` : `usuarios.perfil_${p}`)}</h2>
              {grupo.map((e) => <a key={e.ruta} href={e.ruta} aria-current={ruta === e.ruta || (e === entradaPaquetes && rutasPaquetes.includes(ruta ?? '')) || pantallas.some((p) => p.ruta === ruta && p.menuDe === e.ruta) ? paginaActual : undefined} onClick={(evento) => {
                if (evento.button !== 0 || evento.ctrlKey || evento.metaKey || evento.shiftKey || evento.altKey || !navegar) return;
                evento.preventDefault(); navegar(e.ruta);
              }}>{t(e === entradaPaquetes ? 'paquetes.titulo' : e.clave)}</a>)}
            </section>;
          })}
        </nav>
      </aside>}
      <main id="contenido" tabIndex={-1} className={acceso?.estado === 'activo' ? 'contenido-trabajo' : 'contenido-publico'}>{children}</main>
    </div>
  </div>;
}

export function Preferencias({ t, idioma, idiomas, tema, temas, cambiarIdioma, cambiarTema }: {
  t: Textos; idioma: string; idiomas: { codigo: string; nombre: string }[];
  tema: string; temas: { identificador: string; nombre: Record<string, string> }[];
  cambiarIdioma: (valor: string) => void; cambiarTema: (valor: string) => void;
}) {
  const detalle = useRef<HTMLDetailsElement>(null);
  const resumen = useRef<HTMLElement>(null);
  function cerrar() { if (detalle.current) detalle.current.open = false; resumen.current?.focus(); }
  return <details className="preferencias" ref={detalle} onKeyDown={(evento) => { if (evento.key === 'Escape' && detalle.current?.open) { evento.preventDefault(); evento.stopPropagation(); cerrar(); } }}
    onBlur={(evento) => { if (!evento.currentTarget.contains(evento.relatedTarget) && !evento.currentTarget.closest('.panel-persona')?.contains(evento.relatedTarget) && detalle.current) detalle.current.open = false; }}>
    <summary ref={resumen}>{t('comun.preferencias')}</summary>
    <div className="panel-preferencias">
      <label htmlFor="idioma">{t('comun.idioma')}</label>
      <select id="idioma" value={idioma} onChange={(evento) => cambiarIdioma(evento.target.value)}>
        {idiomas.map((opcion) => <option key={opcion.codigo} value={opcion.codigo} lang={opcion.codigo}>{opcion.nombre}</option>)}
      </select>
      <label htmlFor="tema">{t('comun.tema')}</label>
      <select id="tema" value={tema} onChange={(evento) => cambiarTema(evento.target.value)}>
        {temas.map((opcion) => <option key={opcion.identificador} value={opcion.identificador}>{opcion.nombre[idioma] ?? Object.values(opcion.nombre)[0]}</option>)}
      </select>
    </div>
  </details>;
}
