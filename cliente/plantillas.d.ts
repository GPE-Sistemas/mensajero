// Generado por `make generar` desde plantillas/ y sistemas.json. No editar.

export interface Plantillas {
  'gas': {
    'cambio-password': { codigo: string; link: string };
    'nuevo-usuario': { link: string; password: string; usuario: string };
    'reset-password': { link: string };
    'scada-fuera-limite': { fecha: string; limite: string; punto: string; valor: string; valorLimite: string; variable: string };
    'scada-reestablecido': { fecha: string; punto: string; valor: string; variable: string };
  };
}

export interface Remitentes {
  'gas': 'alertas' | 'default';
}
