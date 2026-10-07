import type { Plantillas, Remitentes } from './plantillas';

export type { Plantillas, Remitentes };

export type Sistema = keyof Plantillas & keyof Remitentes;

export interface OpcionesEnvio<S extends Sistema, D> {
  para: string;
  datos: D;
  remitente?: Remitentes[S];
  nombre?: string;
  responderA?: string;
}

/** status 0: no se llegó al mensajero. 502: Postfix no respondió. En esos dos tiene sentido reintentar. */
export declare class MensajeroError extends Error {
  readonly status: number;
}

export declare class Mensajero<S extends Sistema> {
  constructor(opciones: { url: string; apikey: string; timeoutMs?: number });
  enviar<P extends keyof Plantillas[S] & string>(
    plantilla: P,
    opciones: OpcionesEnvio<S, Plantillas[S][P]>,
  ): Promise<{ id: string }>;
}
