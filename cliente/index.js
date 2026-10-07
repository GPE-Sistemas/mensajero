'use strict';

/** status 0: no se llegó al mensajero. 502: Postfix no respondió. En esos dos tiene sentido reintentar. */
class MensajeroError extends Error {
  constructor(status, mensaje) {
    super(mensaje);
    this.name = 'MensajeroError';
    this.status = status;
  }
}

class Mensajero {
  constructor({ url, apikey, timeoutMs = 15000 }) {
    this.url = url.replace(/\/+$/, '');
    this.apikey = apikey;
    this.timeoutMs = timeoutMs;
  }

  async enviar(plantilla, { para, datos, remitente, nombre, responderA }) {
    let res;
    try {
      res = await fetch(`${this.url}/v1/email`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${this.apikey}` },
        body: JSON.stringify({ plantilla, para, datos, remitente, nombre, responderA }),
        signal: AbortSignal.timeout(this.timeoutMs),
      });
    } catch (err) {
      throw new MensajeroError(0, `no se pudo llegar al mensajero: ${err.message}`);
    }
    const cuerpo = await res.json().catch(() => ({}));
    if (!res.ok) {
      throw new MensajeroError(res.status, cuerpo.error || res.statusText);
    }
    return { id: cuerpo.id };
  }
}

module.exports = { Mensajero, MensajeroError };
